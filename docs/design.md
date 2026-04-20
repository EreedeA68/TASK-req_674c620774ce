# Keystone Offline Operations Management — Architecture & Design

## System Architecture Overview

Keystone runs entirely on-premise with no external internet dependencies. The frontend is a React single-page application that communicates with a Go/Echo REST API over the local network or localhost. The Echo backend connects to a PostgreSQL database for all persistent state. The entire system is packaged with Docker Compose for single-command deployment.

```
+-------------------------------------------------------------------+
|                          Docker Compose                           |
|                                                                   |
|  +----------------------+         +---------------------------+  |
|  |   React SPA          |  HTTP   |   Go / Echo API           |  |
|  |   (port 3000)        |<------->|   (port 8080)             |  |
|  |                      |  REST   |                           |  |
|  |  Role-based nav      |         |  Auth & MFA handlers      |  |
|  |  Candidate intake    |         |  Candidate service        |  |
|  |  Lost & Found UI     |         |  Listing service          |  |
|  |  Parts catalog UI    |         |  Parts service            |  |
|  |  Search & reports    |         |  Search service           |  |
|  |  Audit viewer        |         |  Audit service            |  |
|  +----------------------+         |  Document service         |  |
|                                   +---------------------------+  |
|                                              |                    |
|                                   +---------------------------+  |
|                                   |   PostgreSQL              |  |
|                                   |                           |  |
|                                   |  users / sessions         |  |
|                                   |  candidates / documents   |  |
|                                   |  listings                 |  |
|                                   |  parts / part_versions    |  |
|                                   |  audit_logs               |  |
|                                   +---------------------------+  |
+-------------------------------------------------------------------+
         ^
         | HTTP (local network / localhost)
         |
   Admin workstations / Specialist terminals / Clerk stations
```

**Request flow:** A browser on the local network loads the React SPA. Each user action calls a JSON REST endpoint on the Echo API. Echo validates the JWT, checks RBAC permissions, runs business logic, and returns a JSON response. PostgreSQL is the sole persistent store; no external services, webhooks, or cloud APIs are used.

Optional OpenAPI documentation is generated locally from the Echo routes and served at `/api/docs` when enabled in configuration.

---

## Role Definitions

| Role | Description |
|------|-------------|
| **Administrator** | Full system access. Manages user accounts, role assignments, system configuration, and access control. Can perform all actions of every other role. |
| **Intake Specialist** | Creates and manages candidate profiles, uploads supporting documents, and submits completed profiles for review. Cannot approve or reject. |
| **Reviewer** | Processes approval workflows for candidate submissions. Approves or rejects with required comments. Can override duplicate-post flags on lost-and-found listings. |
| **Inventory Clerk** | Manages lost-and-found listings (publish, edit, unlist, delete) and maintains the automotive parts catalog including fitment data, OEM mappings, attribute templates, and bulk imports. |
| **Auditor** | Read-only oversight role. Can view all records, audit logs, and reports but cannot create, modify, or delete any data. Cannot export without explicit download permission. |

---

## Core Modules

### Auth & MFA Module
Handles local username/password authentication only — no OAuth, SSO, or external identity providers. Passwords must be at least 12 characters and are hashed with bcrypt. JWTs are issued on successful login, signed with a server-side HMAC-SHA256 secret, and validated on every protected request. Brute-force protection locks accounts for 15 minutes after 5 consecutive failed login attempts. Optional MFA is implemented via TOTP (RFC 6238): a shared secret is generated locally and stored AES-256 encrypted in the database; the user registers it with an authenticator app and subsequent logins require a valid 6-digit TOTP code. No SMS or email is used at any point.

### Candidate Intake Module
Intake Specialists create candidate profiles capturing: basic demographics, initial exam scores, application details, transfer preferences, and a document checklist. Profiles begin in DRAFT status. A profile may only be submitted for review once it reaches the required completeness status (all mandatory checklist items present and all required fields populated). Document uploads are restricted to PDF, JPG, and PNG formats with a 20 MB per-file limit. Each uploaded file is SHA-256 hashed on receipt; the hash is stored and re-verified on every download. The review workflow proceeds: DRAFT → SUBMITTED → APPROVED or REJECTED. Reviewers must supply a comment when rejecting. Status changes are timestamped in MM/DD/YYYY hh:mm AM/PM format.

### Lost & Found Module
Inventory Clerks publish lost-and-found listings. Each listing must include a category tag, a validated US-style location description (city, state, optional street), and a time window (date/time the item was found or reported). Listings support four lifecycle actions: publish, edit, unlist, and delete. Posts older than 90 days are automatically unlisted by a background job but remain in the database and are searchable for audit purposes. Duplicate detection runs on every new submission: if another listing with the same category and a title similarity score above the configured threshold (default: cosine similarity ≥ 0.85) exists within the prior 24 hours, the submission is flagged as a probable duplicate and held pending reviewer override before it can be published.

### Automotive Parts Module
Inventory Clerks manage a parts catalog with the following data per part: fitment dimensions (vehicle model, year, engine, transmission), OEM part number to alternative part number mappings, and attribute templates for structured additional properties. Each edit to a part record creates a new immutable version; prior versions are retained in full and are viewable and comparable side-by-side before a new version is promoted to active. Bulk CSV import provides a preview-and-validate step showing all rows and detected errors before commit; imports execute inside a single database transaction and fail atomically if any row violates schema or constraint rules. Bulk export allows configurable field selection.

### Search & Reporting Module
A unified search endpoint accepts combined filters across candidates, listings, and parts. Optional fuzzy matching is available as a query parameter. Configurable exports allow field selection before download. KPI dashboards expose: candidate conversion rate (submitted → approved), review cycle time (submission to decision), and quota utilization per category. Sensitive fields (e.g., demographics, exam scores) are masked by default in search results and exports unless the requesting user holds explicit view-sensitive permission.

### Audit Logging Module
An append-only audit log records every significant action: logins and logouts, CRUD operations on all entities, approval and rejection events, export events, and document download events. Each entry captures: actor user ID, actor role, timestamp (UTC stored, displayed in MM/DD/YYYY 12-hour format), device identifier from the request, action type, target entity type and ID, and a JSON diff blob with before and after values. The audit log table has no UPDATE or DELETE privileges at the database level. Auditors and Administrators can query the audit log via the API.

### Document Management Module
Documents are uploaded via multipart form data, stored on local disk under a content-addressed path derived from the SHA-256 hash of the file contents, and linked to their parent candidate record in the database. On each download request, the backend re-computes the SHA-256 hash of the stored file and compares it to the stored hash; a mismatch aborts the download and triggers an audit alert. Download rights are a separately granted permission; granting and revoking download rights are both audit-logged. Optional watermarking embeds the requesting user's name and timestamp into generated PDF download copies.

---

## Database Design Summary

### users
`id`, `username` (unique), `password_hash` (bcrypt), `role` (ADMIN / INTAKE_SPECIALIST / REVIEWER / INVENTORY_CLERK / AUDITOR), `status` (ACTIVE / LOCKED), `locked_until`, `failed_login_count`, `mfa_enabled` (boolean), `mfa_secret_encrypted`, `created_at`, `updated_at`

### sessions
`id`, `user_id` (FK → users), `token_hash`, `created_at`, `expires_at`, `revoked`

### login_attempts
`id`, `user_id` (FK → users), `attempted_at`, `success` (boolean), `ip_address`, `device_id`

### permissions
`id`, `user_id` (FK → users), `permission` (CREATE / APPROVE / EXPORT / DOWNLOAD / VIEW_SENSITIVE), `granted_by` (FK → users), `granted_at`, `revoked_at`

### candidates
`id`, `first_name`, `last_name`, `date_of_birth`, `gender`, `contact_info` (jsonb), `exam_scores` (jsonb), `application_details` (jsonb), `transfer_preferences` (jsonb), `status` (DRAFT / SUBMITTED / APPROVED / REJECTED), `completeness_status` (INCOMPLETE / COMPLETE), `created_by` (FK → users), `reviewed_by` (FK → users, nullable), `review_comment`, `created_at`, `submitted_at`, `reviewed_at`, `updated_at`

### candidate_documents
`id`, `candidate_id` (FK → candidates), `document_type`, `file_name`, `file_path`, `file_size`, `mime_type`, `sha256_hash`, `uploaded_by` (FK → users), `uploaded_at`

### document_checklist_items
`id`, `candidate_id` (FK → candidates), `item_name`, `required` (boolean), `fulfilled` (boolean), `document_id` (FK → candidate_documents, nullable)

### listings
`id`, `title`, `description`, `category` (FK → listing_categories), `location_description`, `location_city`, `location_state`, `found_at` (timestamptz), `status` (PUBLISHED / UNLISTED / DELETED), `duplicate_flag` (boolean), `duplicate_of_id` (FK → listings, nullable), `override_by` (FK → users, nullable), `override_at`, `auto_unlisted` (boolean), `auto_unlisted_at`, `created_by` (FK → users), `created_at`, `updated_at`

### listing_categories
`id`, `name` (unique), `description`

### parts
`id`, `part_number` (unique), `name`, `description`, `active_version_id` (FK → part_versions, nullable), `created_by` (FK → users), `created_at`

### part_versions
`id`, `part_id` (FK → parts), `version_number`, `fitment` (jsonb: model, year, engine, transmission), `oem_number`, `alternative_numbers` (jsonb array), `attributes` (jsonb), `promoted_by` (FK → users, nullable), `promoted_at`, `created_by` (FK → users), `created_at`

### bulk_import_jobs
`id`, `entity_type` (PARTS), `file_name`, `total_rows`, `valid_rows`, `error_rows`, `status` (PREVIEW / COMMITTED / FAILED), `errors_json`, `created_by` (FK → users), `created_at`, `committed_at`

### audit_logs
`id`, `actor_id` (FK → users, nullable for system), `actor_role`, `action`, `target_type`, `target_id`, `diff_json`, `ip_address`, `device_id`, `occurred_at`

### encrypted_credentials
`id`, `owner_id`, `owner_type`, `field_name`, `encrypted_value`, `key_version`, `created_at`

**Relationships summary:**
- `users` ← `sessions`, `login_attempts`, `permissions`, `candidates` (as creator), `candidates` (as reviewer), `listings` (as creator), `part_versions` (as creator/promoter), `audit_logs` (as actor)
- `candidates` ← `candidate_documents`, `document_checklist_items`
- `listings` ← `listing_categories`
- `parts` ← `part_versions`
- `bulk_import_jobs` links to `parts` indirectly via committed rows

---

## Security Design

### Password Policy
Minimum 12 characters. Passwords are hashed using bcrypt with a per-user salt. Raw passwords are never stored or logged.

### AES-256 Encryption at Rest
Sensitive secrets (TOTP shared secrets, any stored credential values) are encrypted using AES-256-GCM before storage in `encrypted_credentials`. Encryption keys are versioned; the key version is stored alongside the ciphertext to support rotation.

### JWT Authentication
On successful login (and TOTP verification if MFA is enabled), a JWT is issued containing user ID, role, and expiry. The token is signed with HMAC-SHA256. All protected Echo routes validate the JWT via middleware before executing any handler.

### RBAC and Fine-Grained Permissions
Role-based access control enforces which menu items are visible and which API routes are accessible per role. Overlay permissions (CREATE, APPROVE, EXPORT, DOWNLOAD, VIEW_SENSITIVE) are stored per user in the `permissions` table and checked at the handler level, allowing Administrators to grant or revoke individual capabilities independent of role.

### TOTP-Based MFA
When a user enables MFA, the backend generates a TOTP shared secret using a CSPRNG, encrypts it with AES-256, stores it in `encrypted_credentials`, and returns the raw secret once to the client for QR code enrollment. On subsequent logins the user submits a 6-digit TOTP code; the server decrypts the secret and validates the code against the current and adjacent time windows (±30 seconds) per RFC 6238. No SMS or email is involved.

### Brute-Force Lockout
After 5 consecutive failed login attempts, `users.status` is set to LOCKED and `users.locked_until` is set 15 minutes into the future. All subsequent attempts during the lockout period return HTTP 403. Lockout events are written to `audit_logs`. Administrators can manually unlock accounts.

### Document Integrity: SHA-256 Verification
On upload, the backend computes SHA-256 of the file bytes and stores it in `candidate_documents.sha256_hash`. On every download request, the stored file bytes are re-hashed and compared to the stored value. If the hash does not match, the download is aborted and an audit log entry is created with action `DOCUMENT_INTEGRITY_FAILURE`.

### Optional Watermarking
When a document is downloaded, the backend can generate a watermarked copy embedding the requesting user's display name and the download timestamp in the PDF margin. The original stored file is never modified; watermarking is applied to the download stream only.

### Download Permissions and Logging
The DOWNLOAD permission is a separately granted overlay permission. Attempting to download a document without it returns HTTP 403. Every successful download is written to `audit_logs` with action `DOCUMENT_DOWNLOAD`, including the document ID, candidate ID, actor, timestamp, and device identifier.

### Sensitive Data Masking
Fields designated as sensitive (demographics, exam scores, government IDs) are masked in API responses and exports unless the requesting user has the VIEW_SENSITIVE permission. Masked values use the format `****` or `****XXXX` (last 4 characters visible for identifiers).

---

## Duplicate Detection Design

On every new listing submission, the backend queries all listings with the same category created within the prior 24 hours that are in PUBLISHED or SUBMITTED status. For each candidate duplicate, it computes cosine similarity between the TF-IDF vectors of the new title and the existing title. If any similarity score meets or exceeds the configured threshold (default 0.85), the new listing is saved with `duplicate_flag = true` and `duplicate_of_id` pointing to the matched listing, and its status is held as PENDING_REVIEW rather than PUBLISHED. A Reviewer must explicitly call the override endpoint, providing a comment, before the listing is published. The override actor and timestamp are recorded in `listings.override_by` and `listings.override_at`, and the event is audit-logged.

---

## Auto-Unlist Rule

A background job (configurable cron, default: runs nightly) queries all listings with `status = PUBLISHED` and `created_at < NOW() - INTERVAL '90 days'`. Each matched listing is updated to `status = UNLISTED` with `auto_unlisted = true` and `auto_unlisted_at = NOW()`. The listing row is retained in the database and remains accessible via the search API (with appropriate filters). Auditors and Administrators can query auto-unlisted listings. Authors are not externally notified (no email or SMS), but the status change is reflected in the UI on next load and is recorded in the audit log.

---

## Versioning Design

### Immutable Part Versions
Every mutation to a part's fitment, OEM number, alternative numbers, or attributes creates a new row in `part_versions` rather than updating the existing row. The new version's `version_number` is incremented. The prior version row is never modified. `parts.active_version_id` points to the currently promoted version. Users can retrieve the full version history via `GET /api/parts/{id}/versions` and compare any two versions side-by-side using the response's diff fields before calling the promote endpoint.

### Before/After Diffs in Audit Logs
Every CRUD action that modifies an entity serializes the before-state and after-state to JSON and stores them in `audit_logs.diff_json`. For part version promotions, the diff captures the full field-level delta between the outgoing active version and the newly promoted version.

---

## Bulk Import Design

### CSV Preview-and-Validate Step
A bulk import begins with `POST /api/parts/import` with the CSV file. The backend parses every row, validates schema (required columns present, data types correct) and constraint rules (part number uniqueness, fitment format), and returns a preview response listing all rows with per-row validation results and a summary of valid vs. error row counts. No data is written to the database at this stage.

### Atomic Commit
After the user reviews the preview, a second call commits the import. The backend opens a single PostgreSQL transaction, inserts all valid rows (or all rows if zero errors), and commits. If any row raises a constraint error during the transaction, the entire transaction is rolled back and the import job is marked FAILED with the error detail. There is no partial commit: either all rows succeed or none do. The import job record in `bulk_import_jobs` captures the outcome, row counts, and any error JSON.

---
