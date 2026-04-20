# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the full source tree
COPY . .

# Build the binary with production flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o keystone-backend \
    ./cmd/server

# =============================================================================
# Stage 2: Test runner (used by run_tests.sh via docker build --target test)
# =============================================================================
FROM builder AS test
ENV CGO_ENABLED=0
CMD ["go", "test", "./...", "-cover", "-coverprofile=/tmp/coverage.out", "-timeout", "120s"]

# =============================================================================
# Stage 3: Runtime
# =============================================================================
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user for security
RUN addgroup -S keystone && adduser -S keystone -G keystone

WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/keystone-backend .

# Create documents directory and set ownership
RUN mkdir -p /app/documents && chown -R keystone:keystone /app

# Switch to non-root user
USER keystone

EXPOSE 8080

CMD ["./keystone-backend"]
