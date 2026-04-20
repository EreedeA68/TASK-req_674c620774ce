#!/usr/bin/env bash
set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0
BACKEND_COVERAGE=0
FRONTEND_COVERAGE=0
E2E_PASS=false

log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASS++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((FAIL++)); }
log_info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

# Helper: compare floats without bc — uses awk as a fallback
float_lt() {
  # float_lt A B  => returns 0 (true) if A < B
  local a="$1"
  local b="$2"
  if command -v bc &>/dev/null; then
    (( $(echo "$a < $b" | bc -l) ))
  else
    awk -v a="$a" -v b="$b" 'BEGIN { exit !(a < b) }'
  fi
}

# 1. Build Docker images
log_info "Building Docker images..."
docker-compose build 2>&1 | tail -5 || { log_fail "Docker build failed"; exit 1; }
log_info "Docker build complete"

# 2. Start services
log_info "Starting services..."
docker-compose up -d db backend frontend 2>&1

# 3. Wait for backend health
log_info "Waiting for backend health..."
BACKEND_READY=false
for i in $(seq 1 60); do
  if curl -sf http://localhost:8080/api/health > /dev/null 2>&1; then
    BACKEND_READY=true
    break
  fi
  sleep 1
done
if [ "$BACKEND_READY" = false ]; then
  log_fail "Backend did not become healthy within 60s"
  docker-compose logs backend | tail -30
  docker-compose down -v
  exit 1
fi
log_pass "Backend healthy"

# 4. Wait for frontend
log_info "Waiting for frontend..."
FRONTEND_READY=false
for i in $(seq 1 60); do
  if curl -sf http://localhost:3000 > /dev/null 2>&1; then
    FRONTEND_READY=true
    break
  fi
  sleep 1
done
if [ "$FRONTEND_READY" = false ]; then
  log_fail "Frontend did not become available within 60s"
  docker-compose logs frontend | tail -20
  docker-compose down -v
  exit 1
fi
log_pass "Frontend available"

# 5. Backend unit + integration tests (run in the builder stage which has the Go toolchain)
log_info "Building backend test image..."
docker build --target test -t keystone-test-backend -f backend/Dockerfile backend/ 2>&1 || { log_fail "Backend test image build failed"; BACKEND_COVERAGE=0; }

log_info "Running backend tests..."
if docker run --rm \
  -e JWT_SECRET="${JWT_SECRET:-test-secret-32-chars-padding-ok!}" \
  -e AES_KEY="${AES_KEY:-12345678901234567890123456789012}" \
  -e TEST_DB_DSN="postgres://${DB_USER:-postgres}:${DB_PASSWORD:-postgres}@${DB_HOST:-localhost}:${DB_PORT:-5432}/${TEST_DB_NAME:-keystone_test}?sslmode=disable" \
  --network host \
  keystone-test-backend 2>&1 | tee /tmp/backend_test_output.txt; then
  log_pass "Backend tests passed"
  COVERAGE_OUT=$(grep "^total:" /tmp/backend_test_output.txt 2>/dev/null | awk '{print $3}' | tr -d '%')
  BACKEND_COVERAGE=${COVERAGE_OUT:-0}
  log_info "Backend coverage: ${BACKEND_COVERAGE}%"
else
  log_fail "Backend tests failed"
  BACKEND_COVERAGE=0
fi

# 6. Frontend tests (run in the builder stage which has Node.js)
log_info "Building frontend test image..."
docker build --target test -t keystone-test-frontend -f frontend/Dockerfile frontend/ 2>&1 || { log_fail "Frontend test image build failed"; FRONTEND_COVERAGE=0; }

log_info "Running frontend component tests..."
if docker run --rm keystone-test-frontend 2>&1 | tee /tmp/frontend_test_output.txt; then
  log_pass "Frontend tests passed"
  FRONTEND_COVERAGE=$(grep "All files" /tmp/frontend_test_output.txt 2>/dev/null | awk '{print $10}' | tr -d '%' || echo "0")
  FRONTEND_COVERAGE=${FRONTEND_COVERAGE:-0}
  log_info "Frontend coverage: ${FRONTEND_COVERAGE}%"
else
  log_fail "Frontend tests failed"
  FRONTEND_COVERAGE=0
fi

# 7. E2E tests
log_info "Running E2E tests..."
cd tests/e2e
npm install --quiet 2>/dev/null || true
if npx playwright install chromium --with-deps 2>/dev/null && npx playwright test --config=playwright.config.js 2>&1; then
  log_pass "E2E tests passed"
  E2E_PASS=true
else
  log_fail "E2E tests failed"
fi
cd ../..

# 8. Tear down
log_info "Tearing down..."
docker-compose down -v 2>&1

# 9. Print summary
echo ""
echo "=================================================="
echo "                 TEST SUMMARY"
echo "=================================================="
echo -e "Backend coverage:   ${BACKEND_COVERAGE}%  (threshold: 80%)"
echo -e "Frontend coverage:  ${FRONTEND_COVERAGE}% (threshold: 75%)"
echo -e "Tests passed: ${PASS}  Tests failed: ${FAIL}"
echo "=================================================="

# 10. Check exit conditions
COVERAGE_PASS=true

if float_lt "${BACKEND_COVERAGE:-0}" "80"; then
  echo -e "${RED}Backend coverage ${BACKEND_COVERAGE}% is below 80% threshold${NC}"
  COVERAGE_PASS=false
fi

if float_lt "${FRONTEND_COVERAGE:-0}" "75"; then
  echo -e "${RED}Frontend coverage ${FRONTEND_COVERAGE}% is below 75% threshold${NC}"
  COVERAGE_PASS=false
fi

if [ "$FAIL" -eq 0 ] && [ "$COVERAGE_PASS" = true ] && [ "$E2E_PASS" = true ]; then
  echo -e "${GREEN}ALL TESTS PASSED${NC}"
  exit 0
else
  echo -e "${RED}SOME TESTS FAILED${NC}"
  exit 1
fi
