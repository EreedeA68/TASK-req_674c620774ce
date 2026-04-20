# Keystone Delivery Acceptance and Project Architecture Audit (Static-Only)

Date: 2026-04-20
Reviewer Mode: Static analysis only (no runtime execution)

## 1. Verdict
- Overall conclusion: **Fail**

The repository is substantial and implements many prompt-aligned features, but there are material security and requirement-fit gaps (notably object/data-scope authorization and missing candidate checklist workflow) that prevent delivery acceptance.

## 2. Scope and Static Verification Boundary
- Reviewed:
  - Documentation and run/test instructions
  - Backend entrypoint, route registration, middleware, auth/authz, core modules (candidate, listing, parts, reports, search, documents, audit)
  - DB schema and constraints
  - Unit/integration/frontend/e2e test sources (static review only)
  - Frontend route/menu/role guards and key workflow pages
- Not reviewed:
  - Runtime behavior under real DB/network/container/browser execution
  - Performance characteristics, concurrency behavior under load
  - Actual coverage percentages reported by CI/runtime
- Intentionally not executed:
  - Application startup
  - Docker/Compose
  - Tests (unit/integration/frontend/e2e)
- Claims requiring manual verification:
  - End-to-end runtime flows, cron execution timing, browser rendering fidelity
  - Transaction rollback behavior under live DB failures
  - Real export/download behavior across large datasets and binary edge cases

## 3. Repository / Requirement Mapping Summary
- Prompt core goal mapped: offline on-prem system combining candidate intake, lost-and-found, and automotive parts catalog with RBAC, auditing, reporting, and secure document handling.
- Main implementation areas mapped:
  - Backend Echo APIs and middleware: `backend/cmd/server/main.go`
  - Core services: `backend/internal/{auth,candidate,lostfound,parts,search,reports,documents,audit}`
  - DB schema/constraints/seeds: `db/init.sql`
  - Frontend role-based React UI: `frontend/src/{App.js,components,pages}`
  - Tests across backend/frontend/e2e: `backend/tests`, `tests/frontend`, `tests/e2e`
- Primary misalignment themes:
  - Insufficient object-level and tenant/site data-scope enforcement
  - Missing candidate document checklist requirement
  - Reporting/export and validation depth weaker than prompt intent

## 4. Section-by-section Review

### 1. Hard Gates
#### 1.1 Documentation and static verifiability
- Conclusion: **Pass**
- Rationale: README provides startup and test commands; project has clear entry points and module structure.
- Evidence: `README.md:12`, `README.md:60`, `README.md:75`, `README.md:82`, `backend/cmd/server/main.go:22`, `backend/cmd/server/main.go:130`, `run_tests.sh:34`, `run_tests.sh:39`

#### 1.2 Material deviation from Prompt
- Conclusion: **Fail**
- Rationale: Core constraints around scoped authorization and candidate checklist completeness are not fully implemented.
- Evidence:
  - Missing checklist model/workflow: `backend/internal/candidate/models.go:6`, `db/init.sql:47`, `db/init.sql:55`, `backend/internal/candidate/service.go:343`
  - Data-scope gaps (object/tenant): `backend/internal/candidate/service.go:95`, `backend/internal/candidate/service.go:100`, `backend/internal/parts/repository.go:28`, `backend/internal/lostfound/repository.go:30`, `backend/internal/search/service.go:56`, `backend/internal/search/service.go:74`, `backend/internal/search/service.go:135`

### 2. Delivery Completeness
#### 2.1 Core requirements coverage
- Conclusion: **Partial Pass**
- Rationale:
  - Implemented: login/lockout/MFA, RBAC routes, candidate draft-submit-approve-reject, listing duplicate flag/override, part versioning/compare/promote, CSV import/export, audit log, download permission/hash check.
  - Missing/weak: document checklist requirement, strict US location validation, robust tenant/object scope enforcement.
- Evidence: `backend/internal/auth/service.go:19`, `backend/internal/auth/service.go:20`, `backend/internal/candidate/service.go:141`, `backend/internal/lostfound/service.go:87`, `backend/internal/lostfound/service.go:92`, `backend/internal/parts/service.go:147`, `backend/internal/parts/handler.go:199`, `backend/internal/documents/service.go:83`, `backend/internal/candidate/service.go:343`, `backend/internal/lostfound/service.go:62`

#### 2.2 Basic end-to-end deliverable (0->1)
- Conclusion: **Pass**
- Rationale: Complete full-stack structure exists with backend/frontend/db/tests/docs; not a code fragment/demo-only layout.
- Evidence: `README.md:12`, `backend/cmd/server/main.go:1`, `frontend/src/App.js:1`, `db/init.sql:1`, `backend/tests/integration/auth_handler_test.go:1`, `tests/frontend/LoginPage.test.js:1`, `tests/e2e/keystone.spec.js:1`

### 3. Engineering and Architecture Quality
#### 3.1 Structure and module decomposition
- Conclusion: **Pass**
- Rationale: Reasonable modular decomposition by domain/service/repository/handler; clear frontend page/component split.
- Evidence: `README.md:12`, `backend/internal/candidate/service.go:1`, `backend/internal/parts/repository.go:1`, `backend/internal/lostfound/handler.go:1`, `frontend/src/pages/candidates/CandidateDetailPage.js:1`, `frontend/src/components/Navbar.js:1`

#### 3.2 Maintainability/extensibility
- Conclusion: **Partial Pass**
- Rationale: Generally maintainable, but several permission/data-scope rules are hardcoded or inconsistently enforced across list vs detail/search/document endpoints.
- Evidence: `backend/internal/candidate/handler.go:73`, `backend/internal/candidate/handler.go:76`, `backend/internal/candidate/service.go:100`, `backend/internal/search/handler.go:39`, `backend/internal/search/handler.go:40`, `backend/cmd/server/main.go:176`

### 4. Engineering Details and Professionalism
#### 4.1 Error handling/logging/validation/API design
- Conclusion: **Partial Pass**
- Rationale:
  - Positive: Consistent response envelope, security middleware, audit hooks, file hash validation.
  - Gaps: weak US-location validation and incomplete object-scope checks.
- Evidence: `backend/internal/middleware/auth.go:39`, `backend/cmd/server/main.go:78`, `backend/internal/audit/service.go:35`, `backend/internal/documents/service.go:82`, `backend/internal/lostfound/service.go:62`, `backend/internal/lostfound/service.go:179`

#### 4.2 Product-like delivery vs demo
- Conclusion: **Partial Pass**
- Rationale: Product-like breadth exists; however, KPI/export masking preview includes hardcoded sample sensitive values and export behavior for unsupported fields degrades to placeholders.
- Evidence: `frontend/src/pages/reports/KPIDashboardPage.js:37`, `frontend/src/pages/reports/KPIDashboardPage.js:41`, `backend/internal/reports/service.go:134`, `backend/internal/reports/service.go:165`

### 5. Prompt Understanding and Requirement Fit
#### 5.1 Business goal and constraint fit
- Conclusion: **Fail**
- Rationale: Key prompt constraints are not fully met:
  - Data-scope isolation (site/org/object) is partial and bypassable on detail/search paths.
  - Candidate checklist requirement is not modeled/enforced.
- Evidence: `backend/internal/candidate/handler.go:73`, `backend/internal/candidate/service.go:95`, `backend/internal/parts/repository.go:28`, `backend/internal/lostfound/repository.go:30`, `backend/internal/search/service.go:74`, `backend/internal/search/service.go:135`, `backend/internal/candidate/models.go:6`, `db/init.sql:47`

### 6. Aesthetics (frontend)
#### 6.1 Visual/interaction quality
- Conclusion: **Partial Pass**
- Rationale: UI has clear sectioning, role-based nav, status/timestamp components, basic feedback states; full visual correctness is runtime-dependent and cannot be fully proven statically.
- Evidence: `frontend/src/components/Navbar.js:5`, `frontend/src/components/StatusBadge.js:1`, `frontend/src/components/TimestampDisplay.js:1`, `frontend/src/pages/candidates/CandidateDetailPage.js:120`, `frontend/src/pages/lostfound/ListingFormPage.js:92`
- Manual verification note: responsive behavior, actual rendering consistency, and interaction transitions require browser execution.

## 5. Issues / Suggestions (Severity-Rated)

### 1) Blocker
- Severity: **Blocker**
- Title: Missing object-level and tenant/site isolation across detail/search paths
- Conclusion: **Fail**
- Evidence:
  - Candidate object access only restricts intake ownership; no org/site checks for other roles: `backend/internal/candidate/service.go:95`, `backend/internal/candidate/service.go:100`
  - Part/listing fetch-by-id ignores org/site scope: `backend/internal/parts/repository.go:28`, `backend/internal/lostfound/repository.go:30`
  - Search queries across whole tables without org/site constraints: `backend/internal/search/service.go:74`, `backend/internal/search/service.go:98`, `backend/internal/search/service.go:135`
- Impact: Cross-scope data exposure/modification risk; violates prompt data-scope constraints and compliance posture.
- Minimum actionable fix:
  - Enforce org/site/object policy in every read/write service method (not only list filters).
  - Add policy helper used by all `Get/Update/Promote/Download/Search` operations.
  - Add deny-by-default for cross-scope records.
- Minimal verification path:
  - Add integration tests proving cross-org/site access returns 403/404 on all detail/search/document routes.

### 2) High
- Severity: **High**
- Title: Candidate documents listing endpoint lacks role restriction
- Conclusion: **Fail**
- Evidence:
  - Route is JWT-only, no `RequireRole`: `backend/cmd/server/main.go:176`
  - Handler relies on `GetCandidate` check: `backend/internal/candidate/handler.go:243`
  - `GetCandidate` only special-cases INTAKE_SPECIALIST ownership, allowing other authenticated roles through: `backend/internal/candidate/service.go:100`
- Impact: Roles outside intended candidate-read scope (e.g., inventory clerk) may enumerate candidate documents.
- Minimum actionable fix:
  - Add explicit role guard on `/api/candidates/:id/documents` route.
  - Add object-scope checks in service for all roles.
- Minimal verification path:
  - Integration test: INVENTORY_CLERK GET `/api/candidates/:id/documents` must return 403.

### 3) High
- Severity: **High**
- Title: Candidate checklist requirement not implemented
- Conclusion: **Fail**
- Evidence:
  - Candidate request/model has demographics/exam/application/transfer only: `backend/internal/candidate/models.go:6`
  - Candidate table lacks checklist fields: `db/init.sql:47`, `db/init.sql:55`
  - Completeness function does not evaluate checklist (or transfer preferences): `backend/internal/candidate/service.go:343`, `backend/internal/candidate/service.go:350`, `backend/internal/candidate/service.go:353`
- Impact: Core business workflow for regulated intake completeness is incomplete.
- Minimum actionable fix:
  - Add checklist schema (required docs/tasks + completion state).
  - Update submission guard to require checklist completeness.
  - Expose checklist CRUD in candidate APIs/UI.
- Minimal verification path:
  - Tests for submit blocked until checklist complete and required document set satisfied.

### 4) High
- Severity: **High**
- Title: Reporting/export implementation is partially placeholder-driven
- Conclusion: **Partial Fail**
- Evidence:
  - Export query only fetches fixed safe columns: `backend/internal/reports/service.go:134`
  - Unsupported requested fields output placeholders `[field]`: `backend/internal/reports/service.go:165`
  - KPI page masking preview uses hardcoded values rather than real export preview: `frontend/src/pages/reports/KPIDashboardPage.js:37`, `frontend/src/pages/reports/KPIDashboardPage.js:41`
- Impact: “Configurable exports with field selection and masked sensitive fields by default” is only partially realized.
- Minimum actionable fix:
  - Implement explicit allowlisted field map with real column extraction/masking rules.
  - Remove placeholder output path; return validation errors for unknown fields.
  - Drive frontend preview from real backend export/preview endpoint.

### 5) Medium
- Severity: **Medium**
- Title: US location validation is weak
- Conclusion: **Partial Fail**
- Evidence:
  - Backend accepts any location containing comma: `backend/internal/lostfound/service.go:62`, `backend/internal/lostfound/service.go:179`
  - Frontend validator only checks minimum length: `frontend/src/pages/lostfound/ListingFormPage.js:7`, `frontend/src/pages/lostfound/ListingFormPage.js:8`
- Impact: Invalid/non-US location strings can pass despite prompt requiring validated US-style locations.
- Minimum actionable fix:
  - Validate `City, ST` (or stronger configured US format) in backend; keep frontend aligned.

### 6) Medium
- Severity: **Medium**
- Title: Watermarking is effectively image-only, not general document-copy watermarking
- Conclusion: **Partial Fail**
- Evidence: Watermark code applies only to JPEG/PNG and no-op otherwise: `backend/internal/documents/service.go:109`, `backend/internal/documents/service.go:110`, `backend/internal/documents/service.go:122`
- Impact: Prompt expectation for optional watermarking on generated download copies is only partially met for common document type (PDF).
- Minimum actionable fix:
  - Add PDF watermark pipeline (or clearly scope/document supported MIME types and enforce at upload/config level).

### 7) Medium
- Severity: **Medium**
- Title: Test suite has critical authorization coverage gaps
- Conclusion: **Partial Fail**
- Evidence:
  - Good authn/authz baseline tests exist: `backend/tests/integration/security_test.go:114`, `backend/tests/integration/security_test.go:210`
  - No tenant/site isolation tests found (no org/site test references in test corpus)
  - No integration test for `/api/candidates/:id/documents` unauthorized role access; route exists in test router too: `backend/tests/integration/helpers_test.go:170`
- Impact: Severe auth defects can remain undetected while tests pass.
- Minimum actionable fix:
  - Add integration tests for object-level and scope-level authorization per resource.

### 8) Low
- Severity: **Low**
- Title: Permissive defaults and CORS increase hardening risk
- Conclusion: **Partial Fail**
- Evidence: wildcard CORS: `backend/cmd/server/main.go:81`; default compose secrets: `docker-compose.yml:36`, `docker-compose.yml:37`
- Impact: Misconfiguration risk in deployments that keep defaults.
- Minimum actionable fix:
  - Fail startup on known-placeholder secrets in non-test mode.
  - Restrict CORS origins to configured local addresses.

## 6. Security Review Summary
- Authentication entry points: **Pass**
  - Evidence: login/logout/me/mfa routes and lockout/MFA logic: `backend/cmd/server/main.go:149`, `backend/internal/auth/service.go:63`, `backend/internal/auth/service.go:19`, `backend/internal/auth/service.go:20`, `backend/internal/auth/service.go:100`
- Route-level authorization: **Partial Pass**
  - Evidence: role/permission middleware on many routes: `backend/cmd/server/main.go:162`, `backend/cmd/server/main.go:192`, `backend/cmd/server/main.go:205`, `backend/cmd/server/main.go:212`
  - Gap: candidate documents list route lacks role guard: `backend/cmd/server/main.go:176`
- Object-level authorization: **Fail**
  - Evidence: id-based fetch/update paths without org/site policy checks: `backend/internal/parts/repository.go:28`, `backend/internal/lostfound/repository.go:30`, `backend/internal/candidate/service.go:100`
- Function-level authorization: **Partial Pass**
  - Evidence: creator/admin checks in candidate update and listing edit: `backend/internal/candidate/service.go:111`, `backend/internal/lostfound/service.go:150`
  - Gap: not consistently applied across all object actions.
- Tenant/user data isolation: **Fail**
  - Evidence: list filters apply org/site: `backend/internal/candidate/handler.go:73`, `backend/internal/lostfound/handler.go:73`, `backend/internal/parts/handler.go:71`
  - But detail/search paths bypass scope: `backend/internal/search/service.go:74`, `backend/internal/search/service.go:135`, `backend/internal/parts/repository.go:28`
- Admin/internal/debug protection: **Pass**
  - Evidence: admin/audit routes role-protected; no obvious unguarded debug endpoints: `backend/cmd/server/main.go:161`, `backend/cmd/server/main.go:166`, `backend/cmd/server/main.go:209`

## 7. Tests and Logging Review
- Unit tests: **Partial Pass**
  - Exists, but many tests are rule replicas vs service-level behavior with mocks/fakes.
  - Evidence: `backend/tests/unit/auth_service_test.go:1`, `backend/tests/unit/candidate_service_test.go:1`, `backend/tests/unit/parts_service_test.go:1`
- API/integration tests: **Partial Pass**
  - Good baseline for auth, role checks, major flows; missing tenant/object-scope and candidate-documents-list auth tests.
  - Evidence: `backend/tests/integration/security_test.go:114`, `backend/tests/integration/candidate_handler_test.go:1`, `backend/tests/integration/helpers_test.go:170`
- Logging categories/observability: **Partial Pass**
  - Request logger + audit event service are present.
  - Evidence: `backend/cmd/server/main.go:78`, `backend/internal/audit/service.go:35`
- Sensitive-data leakage risk in logs/responses: **Partial Pass**
  - Positive: password hash hidden in model JSON tags.
  - Risk: audit before/after snapshots can include full objects and potentially sensitive fields if passed unredacted.
  - Evidence: `backend/internal/db/models.go:14`, `backend/internal/audit/service.go:47`, `backend/internal/audit/service.go:51`

## 8. Test Coverage Assessment (Static Audit)

### 8.1 Test Overview
- Unit tests exist: Go (`backend/tests/unit/*.go`) and SQL-oriented tests (`tests/unit/db_constraints_test.go`).
- API/integration tests exist: Echo integration tests in `backend/tests/integration/*.go`.
- Frontend component tests exist: Jest + RTL in `tests/frontend/*.test.js`.
- E2E tests exist: Playwright in `tests/e2e/keystone.spec.js`.
- Test entry points documented: `README.md:75`, `README.md:82`, `run_tests.sh:34`.
- Framework evidence:
  - Go testing: `backend/tests/integration/auth_handler_test.go:1`, `backend/tests/unit/auth_service_test.go:1`
  - RTL: `tests/frontend/LoginPage.test.js:2`
  - Playwright: `tests/e2e/keystone.spec.js:2`

### 8.2 Coverage Mapping Table
| Requirement / Risk Point | Mapped Test Case(s) | Key Assertion / Fixture / Mock | Coverage Assessment | Gap | Minimum Test Addition |
|---|---|---|---|---|---|
| Login success/failure | `backend/tests/integration/auth_handler_test.go:12`, `backend/tests/integration/auth_handler_test.go:25` | 200 on valid login, 401 on bad password | basically covered | No brute-force timing behavior validation | Add lockout duration boundary tests (14:59 vs 15:01) |
| Lockout after 5 failures | `backend/tests/integration/auth_handler_test.go:157` | Expects 423 on 6th attempt | basically covered | No unlock/retry integration test | Add unlock-after-15-min integration test |
| JWT required on protected routes | `backend/tests/integration/security_test.go:114` | Iterates protected routes expecting 401 | sufficient | None major | Keep as regression suite |
| Role restrictions on key endpoints | `backend/tests/integration/security_test.go:44`, `backend/tests/integration/security_test.go:62`, `backend/tests/integration/security_test.go:157` | 403 on role escalation attempts | basically covered | Missing object-scope role scenarios | Add per-resource object ownership/scope tests |
| Candidate submit requires completeness/docs | `backend/tests/integration/candidate_handler_test.go:113`, `backend/tests/integration/candidate_handler_test.go:145` | Submit fails incomplete; approve flow uploads doc before submit | basically covered | No checklist enforcement tests | Add checklist-required submission tests |
| Lost-found duplicate detection | `backend/tests/integration/listing_handler_test.go:49` | Duplicate listing flagged | basically covered | No strict same-category+24h boundary assertions | Add 24h cutoff and different-category negative tests |
| Parts versioning compare/promote path | `backend/tests/integration/parts_handler_test.go:100`, `backend/tests/integration/parts_handler_test.go:220` | Update creates v2; compare returns diff | basically covered | Promote authorization/object-scope not deeply tested | Add promote unauthorized/cross-scope tests |
| Download permission enforcement | `backend/tests/integration/security_test.go:130` | Route requires auth | insufficient | No allow/deny per `download_permissions` records | Add documents download permission matrix tests |
| Tenant/site isolation | No direct test found | N/A | missing | Severe defect class can pass unnoticed | Add org/site fixture matrix for list/detail/search |
| Candidate documents list authz | No direct test found; route present `backend/tests/integration/helpers_test.go:170` | N/A | missing | Potential unauthorized document enumeration | Add 403 test for INVENTORY_CLERK on `/candidates/:id/documents` |
| Search filtering + fuzzy | `backend/tests/integration/search_handler_test.go:34`, `backend/tests/integration/search_handler_test.go:67` | Query works, fuzzy no-error | insufficient | No scope isolation validation except clerk-candidate type check | Add org/site-scoped search result tests |
| DB immutability/integrity constraints | `tests/unit/db_constraints_test.go:155`, `tests/unit/db_constraints_test.go:558` | Immutable trigger and FK/unique checks | sufficient | Runtime transaction failure-path coverage still absent | Add service-level rollback tests for import failures |

### 8.3 Security Coverage Audit
- Authentication tests: **Basically covered** (login, invalid creds, lockout, invalid token, logout invalidation).
  - Evidence: `backend/tests/integration/auth_handler_test.go:12`, `backend/tests/integration/auth_handler_test.go:157`, `backend/tests/integration/security_test.go:13`, `backend/tests/integration/security_test.go:210`
- Route authorization tests: **Basically covered** for many role gates.
  - Evidence: `backend/tests/integration/security_test.go:44`, `backend/tests/integration/security_test.go:62`, `backend/tests/integration/security_test.go:157`
- Object-level authorization tests: **Insufficient**.
  - Evidence: no tests asserting cross-tenant/cross-owner denial on detail endpoints.
- Tenant/data isolation tests: **Missing**.
  - Evidence: no org/site isolation tests found in test corpus.
- Admin/internal protection tests: **Basically covered**.
  - Evidence: audit/admin access tests: `backend/tests/integration/security_test.go:75`, `backend/tests/integration/security_test.go:100`, `backend/tests/integration/security_test.go:173`

### 8.4 Final Coverage Judgment
- **Fail**

Major authn/authz happy-path and role-gate tests exist, but critical risk areas (object-level authorization, tenant/site isolation, candidate document listing authorization, and permission-specific document download behavior) are uncovered or insufficiently covered. As a result, severe security defects could remain while tests still pass.

## 9. Final Notes
- This audit is static-only and does not claim runtime success.
- Findings are consolidated to root-cause issues to reduce duplication.
- Manual verification is still required for runtime/browser/cron behaviors and operational performance.
