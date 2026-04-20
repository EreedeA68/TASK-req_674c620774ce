# Keystone Offline Operations Management — Design Questions and Answers

## 1. How is duplicate post detection implemented — what similarity algorithm is used?

**Question:** When a new lost-and-found listing is submitted, how does the system determine whether it is a duplicate of an existing listing, and what algorithm drives the similarity comparison?

**Answer:** Duplicate detection runs on every new listing submission. The system queries all existing listings that share the same category tag and were created within the prior 24 hours with a status of PUBLISHED or PENDING_REVIEW. For each candidate match, it computes the **cosine similarity** between TF-IDF (term frequency–inverse document frequency) vectors derived from the title of the new listing and the title of the candidate match. The similarity score ranges from 0.0 (completely different) to 1.0 (identical). If any candidate match produces a score at or above the configured threshold (default: 0.85), the new listing is flagged as a probable duplicate: its status is set to PENDING_REVIEW rather than PUBLISHED, and `duplicate_flag` is set to true with `duplicate_of_id` pointing to the highest-scoring match. The listing cannot be published until a Reviewer explicitly calls the override endpoint with a required comment. The override actor and timestamp are recorded on the listing row and written to the audit log.

---

## 2. What happens when a post reaches 90 days — is the author notified?

**Question:** The system auto-unlists posts older than 90 days. Does the author receive any notification when this happens, and how is the 90-day threshold enforced?

**Answer:** A background job (scheduled nightly by default, configurable via cron expression in application configuration) queries all listings with `status = PUBLISHED` and `created_at` older than 90 days from the current timestamp. Each matched listing is updated: `status` is set to UNLISTED, `auto_unlisted` is set to true, and `auto_unlisted_at` is set to the current UTC timestamp. The change is written to the audit log with action `LISTING_AUTO_UNLISTED`. The author is **not** sent any external notification — the system has no email or SMS capability and no external integrations. The author will observe the status change when they next view their listing in the UI. Auto-unlisted listings remain fully present in the database and are returned in search results when the `include_auto_unlisted=true` query parameter is used, ensuring they remain accessible for audit and compliance review.

---

## 3. How is TOTP MFA stored and validated offline without SMS or email?

**Question:** The system supports optional MFA. How is the shared secret stored securely, and how is a time-based one-time password validated in an environment with no internet, SMS, or email?

**Answer:** When a user enables MFA, the Go backend generates a cryptographically random TOTP shared secret using a CSPRNG and encodes it in base32. The raw secret is returned to the client exactly once (for QR code enrollment in an authenticator app such as Google Authenticator or Authy). Before persisting, the backend encrypts the secret using **AES-256-GCM** with a server-managed key and stores the ciphertext in the `encrypted_credentials` table alongside the key version identifier. No plain-text secret is stored in the database.

On subsequent logins, after the user provides valid credentials, the API returns a short-lived pre-MFA JWT and sets `mfa_required: true`. The client prompts the user for their 6-digit TOTP code. When the client calls `POST /api/auth/mfa/verify`, the server decrypts the stored secret, computes the expected TOTP codes for the current 30-second window and the immediately adjacent windows (one before and one after, to tolerate small clock skew), and compares them to the submitted code. If any window matches, authentication succeeds and a full-access JWT is issued. No network call of any kind is made during this process — the entire validation runs locally using the stored secret and the server's system clock. SMS and email are never used at any point.

---

## 4. What triggers an audit log entry and what fields are captured?

**Question:** The system requires full-chain operational audit logging. Which specific events always produce an audit log entry, and what data is recorded in each entry?

**Answer:** The following events always produce an entry in the `audit_logs` table:

- Successful login and logout
- Failed login attempt (each occurrence, recording the attempted username)
- Account lockout (triggered after 5 consecutive failures)
- Account unlock by an Administrator
- MFA setup or disable
- Candidate profile created, updated, submitted, approved, or rejected
- Document uploaded or downloaded
- Document integrity failure (SHA-256 mismatch on download)
- Lost-and-found listing created, edited, unlisted, deleted, or auto-unlisted
- Duplicate override performed by a Reviewer
- Automotive part created, updated (new version created), or version promoted to active
- Bulk import job previewed or committed
- Search export performed
- KPI report exported
- Permission granted or revoked (DOWNLOAD, EXPORT, etc.)
- User account created, updated, or role changed
- Any configuration change

Each audit log entry records:
- `actor_id` — UUID of the user who performed the action (null for system-generated events)
- `actor_role` — role snapshot at the time of the action
- `action` — action type string (e.g., `CANDIDATE_APPROVED`, `DOCUMENT_DOWNLOAD`)
- `target_type` — entity type string (e.g., `candidate`, `listing`, `part`)
- `target_id` — UUID of the affected entity
- `diff_json` — before and after JSON values for CRUD operations; null for non-mutating events
- `ip_address` — client IP address from the request
- `device_id` — device identifier extracted from the request header
- `occurred_at` — UTC timestamp of the event

The `audit_logs` table has no UPDATE or DELETE privileges at the database level; only INSERT is permitted from the application.

---

## 5. How are immutable part versions stored — full copy or diff?

**Question:** Each edit to a part creates a new version. Is each version stored as a complete copy of all fields, or as a diff from the prior version?

**Answer:** Each version is stored as a **full copy** of all part fields in the `part_versions` table. When an Inventory Clerk edits a part, the backend inserts a new row in `part_versions` containing the complete new state: fitment (model, year, engine, transmission), OEM number, alternative numbers, and attributes. The prior version row is never modified or deleted. The `parts` table holds a foreign key `active_version_id` pointing to whichever version is currently promoted as active. This full-copy approach ensures that any version can be retrieved in its entirety without reconstructing a chain of diffs, simplifies the comparison view (the API can return any two full-version objects and compute the diff on the fly), and guarantees that version records are self-contained and tamper-evident. The `GET /api/parts/{id}/versions` endpoint supports `compare_a` and `compare_b` query parameters that return a field-level diff object computed from the two complete version snapshots.

---

## 6. What happens when a bulk CSV import partially fails?

**Question:** If a CSV file has 100 rows and 3 contain validation errors, does the system commit the 97 valid rows and reject the 3, or does it reject the entire file?

**Answer:** The system rejects the **entire import** atomically — there is no partial commit. The bulk import workflow has two distinct phases:

**Phase 1 — Preview:** The user uploads the CSV via `POST /api/parts/import` with `commit=false`. The backend parses all rows, validates schema (required columns, correct data types) and constraint rules (part number uniqueness, fitment format), and returns a preview response listing every row with per-row validation results. No data is written to the database. The user can review the errors and decide whether to fix the file or proceed if errors are acceptable.

**Phase 2 — Commit:** The user calls the same endpoint with `commit=true`. The backend opens a single PostgreSQL transaction and attempts to insert all rows. If any row raises a constraint violation or validation error during the transaction, the entire transaction is rolled back and the import job record is updated to status FAILED with the error detail JSON. No rows from that file are persisted. The user must correct the CSV and start a new import job. This atomic behavior is intentional: it prevents a partially-imported catalog from being in an inconsistent state and ensures that every committed import is complete and self-consistent.

---

## 7. How is sensitive data masked in exports vs the UI?

**Question:** Sensitive fields such as demographics and exam scores must be masked by default. How does masking differ between the UI, the API, and exported files?

**Answer:** Masking is controlled by the **VIEW_SENSITIVE** permission, which is an overlay permission stored in the `permissions` table and checked at the handler level independently of role.

**In the UI and API responses:** When a user without VIEW_SENSITIVE permission requests a candidate record, fields designated as sensitive (date of birth, government identifiers, exam scores, contact details beyond display name) are replaced with masked values. String fields use `****`; identifiers show the last 4 characters as `****XXXX`. The shape of the JSON response remains identical so the UI can render consistently; only the values differ.

**In exports:** The `GET /api/reports/export` endpoint accepts a `mask_sensitive` query parameter (default: `true`). When true, sensitive fields in the export file are replaced with the same masked values regardless of the requesting user's VIEW_SENSITIVE permission. An Administrator or Auditor with VIEW_SENSITIVE permission can set `mask_sensitive=false` to receive unmasked export data, subject to the EXPORT permission check. Every export event is written to the audit log regardless of masking level.

---

## 8. What defines "completeness" for a candidate profile before submission?

**Question:** Profiles must have completeness_status COMPLETE before they can be submitted for review. What are the exact criteria for completeness?

**Answer:** A candidate profile's `completeness_status` is computed dynamically whenever the profile is read and whenever an update is saved. The profile is considered COMPLETE when all of the following conditions are satisfied:

1. All required top-level fields are non-null and non-empty: `first_name`, `last_name`, `date_of_birth`, `gender`, `contact_info` (with at least email and phone), `application_details` (with at least position and department).
2. `exam_scores` contains at least the required score fields defined in the system configuration (at minimum `written` and `practical`).
3. `transfer_preferences` is present (may be minimal but must not be null).
4. Every `document_checklist_items` row where `required = true` has `fulfilled = true`, meaning a document has been uploaded and linked to that checklist item.

If any of these conditions is not met, `completeness_status` remains INCOMPLETE. The `POST /api/candidates/{id}/submit` endpoint checks this status before transitioning the candidate to SUBMITTED and returns HTTP 400 with a list of the specific unfulfilled conditions if incomplete, so the Intake Specialist knows exactly what to provide.

---

## 9. How are download rights granted and revoked?

**Question:** Download rights are a separately managed permission. How are they granted, what grants them access to, and how are they revoked?

**Answer:** Download rights are controlled by the `DOWNLOAD` permission stored in the `permissions` table as a row per user. Only an Administrator can grant or revoke this permission via user management actions (which call the permission management logic internally — there is no separate public endpoint; the Admin UI calls user update flows).

When DOWNLOAD permission is granted, a row is inserted into `permissions` with `permission = 'DOWNLOAD'`, `granted_by` set to the Administrator's user ID, and `granted_at` set to the current timestamp. The `revoked_at` field is null while the permission is active.

When DOWNLOAD permission is revoked, the existing row's `revoked_at` is set to the current timestamp. The permission check at the handler level queries for a row with `permission = 'DOWNLOAD'`, `user_id = <actor>`, and `revoked_at IS NULL`. A permission with a non-null `revoked_at` is treated as inactive.

Both the grant and revoke actions produce audit log entries with action `PERMISSION_GRANTED` or `PERMISSION_REVOKED`, recording the actor (Administrator), the target user, the permission type, and the timestamp.

Any attempt to download a document without an active DOWNLOAD permission returns HTTP 403. Successful downloads are logged with action `DOCUMENT_DOWNLOAD` regardless of permission level.

---

## 10. What is the SHA-256 verification flow for uploaded documents?

**Question:** Document integrity is protected via SHA-256 hash verification. Walk through the full flow from upload to download.

**Answer:**

**On upload:**
1. The client sends the file via `POST /api/candidates/{id}/documents` as multipart form data.
2. The Echo handler reads the file bytes into memory (up to the 20 MB limit).
3. The backend computes `SHA-256(file_bytes)` using the Go `crypto/sha256` package.
4. The file bytes are written to local disk at a content-addressed path (e.g., `documents/<first2hex>/<sha256hex>`).
5. A row is inserted into `candidate_documents` with `sha256_hash` set to the computed hex digest, `file_path` set to the stored path, and all other metadata.
6. An audit log entry is created with action `DOCUMENT_UPLOADED`.

**On download:**
1. The client calls `GET /api/candidates/{id}/documents/{doc_id}/download`.
2. The handler checks the actor's DOWNLOAD permission; 403 if absent.
3. The handler reads `candidate_documents.sha256_hash` and `file_path` from the database.
4. The backend reads the file bytes from disk at the stored path.
5. It re-computes `SHA-256(file_bytes)` from the bytes just read.
6. It compares the computed digest to the stored `sha256_hash`. If they differ, the download is aborted with HTTP 500 (or a specific integrity error code), and an audit log entry is created with action `DOCUMENT_INTEGRITY_FAILURE` including the document ID and both hash values.
7. If the hashes match and watermarking is enabled for the requesting user, a watermarked PDF copy is generated in memory with the user's display name and the current timestamp embedded in the margin.
8. The file bytes (original or watermarked) are streamed to the client.
9. An audit log entry is created with action `DOCUMENT_DOWNLOAD`.

---

## 11. How does RBAC control menu visibility vs data scope?

**Question:** The system enforces RBAC down to menu visibility and data scope. How do these two layers work and how are they distinguished?

**Answer:** RBAC in Keystone operates at two distinct layers:

**Menu visibility (frontend layer):** The React SPA uses the authenticated user's role (returned in `GET /api/auth/me`) to conditionally render navigation items and UI sections. For example, the "Approval Queue" menu item only renders for REVIEWER and ADMIN roles; the "Parts Catalog" management section only renders for INVENTORY_CLERK and ADMIN; the "Audit Logs" section only renders for ADMIN and AUDITOR. This is a UX convenience: it prevents confusing dead-end routes. It is not a security boundary on its own.

**Data scope and API enforcement (backend layer):** Every Echo handler independently verifies the role and permissions of the JWT bearer before processing the request. Roles define the coarse gate (e.g., only REVIEWER or ADMIN may call `POST /api/candidates/{id}/approve`). Data scope narrows the gate further: INTAKE_SPECIALIST may call `GET /api/candidates/{id}` but the handler filters to only return records where `created_by = actor_id` unless the actor is ADMIN or REVIEWER. AUDITOR may call all read endpoints but every write handler returns 403 for an AUDITOR regardless of the request body.

Overlay permissions (CREATE, APPROVE, EXPORT, DOWNLOAD, VIEW_SENSITIVE) add a third dimension: even a role-permitted handler may return 403 if the specific fine-grained permission is absent. For instance, an INVENTORY_CLERK with no EXPORT permission receives 403 on `GET /api/parts/export` even though their role would otherwise allow parts management.

---

## 12. What happens to reserved/active parts when a new version is promoted?

**Question:** When a Reviewer or Clerk promotes a new version of a part to active, what happens to the previously active version and any records that reference it?

**Answer:** When `POST /api/parts/{id}/promote` is called with a target `version_id`:

1. The backend validates that the `version_id` belongs to the specified `part_id` and is not already the active version.
2. `parts.active_version_id` is updated to point to the newly promoted version.
3. The previously active version row in `part_versions` is **not modified or deleted**. It remains in the table with all its original field values, `promoted_at` timestamp, and `promoted_by` reference. It is accessible via `GET /api/parts/{id}/versions`.
4. Any existing references from other records (e.g., bulk import job rows, audit log entries, fitment lookups) that captured a `version_id` continue to point to the specific version they originally referenced. References are never silently redirected to the new active version.
5. The promote action creates an audit log entry with action `PART_VERSION_PROMOTED`, capturing the `part_id`, the outgoing `active_version_id`, the newly promoted `version_id`, the actor, and a diff showing the field-level changes between the two versions in `diff_json`.

The design ensures that historical records always reflect the exact version of a part that was active at the time of a transaction, supporting compliance review and audit reconstruction without ambiguity.
