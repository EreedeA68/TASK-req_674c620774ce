# Keystone Offline Operations Management — REST API Specification

## General Conventions

- **Base URL:** `http://localhost:8080` (or LAN IP on local network)
- **Content-Type:** `application/json` for all request and response bodies unless noted
- **Authentication:** All endpoints except `/api/auth/login` require `Authorization: Bearer <jwt>` header
- **Timestamps:** All timestamps in responses use `MM/DD/YYYY hh:mm AM/PM` format for display fields; ISO 8601 UTC is used in filter parameters
- **Error response shape:** `{ "error": "<message>", "code": "<ERROR_CODE>" }`
- **Pagination:** List endpoints accept `?page=1&limit=25`; responses include `{ "data": [...], "total": N, "page": N, "limit": N }`

---

## Auth

### POST /api/auth/login
Authenticate a user with local credentials.

**Required role:** None (public)

**Request body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Success 200:**
```json
{
  "token": "string (JWT)",
  "mfa_required": false,
  "user": {
    "id": "uuid",
    "username": "string",
    "role": "ADMIN | INTAKE_SPECIALIST | REVIEWER | INVENTORY_CLERK | AUDITOR"
  }
}
```
If MFA is enabled for the account, `mfa_required` is `true` and `token` is a short-lived pre-MFA token; the client must complete MFA via `/api/auth/mfa/verify` before receiving a full-access token.

**Error cases:**
- `400` — missing username or password
- `401` — invalid credentials
- `403` — account locked (includes `locked_until` field in response)

---

### POST /api/auth/logout
Revoke the current JWT session.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:** none

**Success 200:**
```json
{ "message": "Logged out successfully" }
```

**Error cases:**
- `401` — missing or invalid token

---

### GET /api/auth/me
Return the currently authenticated user's profile and permissions.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Success 200:**
```json
{
  "id": "uuid",
  "username": "string",
  "role": "string",
  "mfa_enabled": false,
  "permissions": ["CREATE", "DOWNLOAD"],
  "created_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `401` — missing or invalid token

---

### POST /api/auth/mfa/setup
Generate a new TOTP shared secret for the authenticated user and return the secret for QR code enrollment. Calling this endpoint replaces any previously configured secret.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:** none

**Success 200:**
```json
{
  "secret": "string (base32-encoded TOTP secret, shown once)",
  "otpauth_url": "string (otpauth:// URI for QR code generation)"
}
```

**Error cases:**
- `401` — missing or invalid token

---

### POST /api/auth/mfa/verify
Verify a TOTP code. Used after initial login when `mfa_required: true` to exchange the pre-MFA token for a full-access JWT.

**Required role:** None (uses pre-MFA token in body)

**Request body:**
```json
{
  "pre_mfa_token": "string",
  "code": "string (6-digit TOTP code)"
}
```

**Success 200:**
```json
{
  "token": "string (full-access JWT)",
  "user": {
    "id": "uuid",
    "username": "string",
    "role": "string"
  }
}
```

**Error cases:**
- `400` — missing fields
- `401` — invalid or expired pre-MFA token, or invalid TOTP code
- `429` — too many failed MFA attempts

---

## Candidates

### POST /api/candidates
Create a new candidate profile in DRAFT status.

**Required role:** INTAKE_SPECIALIST, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:**
```json
{
  "first_name": "string",
  "last_name": "string",
  "date_of_birth": "YYYY-MM-DD",
  "gender": "string",
  "contact_info": {
    "email": "string",
    "phone": "string",
    "address": "string"
  },
  "exam_scores": {
    "written": 0,
    "practical": 0
  },
  "application_details": {
    "position": "string",
    "department": "string"
  },
  "transfer_preferences": {
    "willing_to_relocate": true,
    "preferred_locations": ["string"]
  }
}
```

**Success 201:**
```json
{
  "id": "uuid",
  "status": "DRAFT",
  "completeness_status": "INCOMPLETE",
  "created_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `400` — validation error (missing required fields, invalid date format)
- `401` — unauthorized
- `403` — insufficient role

---

### GET /api/candidates
List candidate profiles with optional filters.

**Required role:** INTAKE_SPECIALIST, REVIEWER, ADMIN, AUDITOR

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:** `status`, `created_by`, `page`, `limit`, `fuzzy` (boolean)

**Success 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "first_name": "string (masked if no VIEW_SENSITIVE)",
      "last_name": "string (masked if no VIEW_SENSITIVE)",
      "status": "DRAFT | SUBMITTED | APPROVED | REJECTED",
      "completeness_status": "INCOMPLETE | COMPLETE",
      "created_at": "MM/DD/YYYY hh:mm AM/PM",
      "submitted_at": "MM/DD/YYYY hh:mm AM/PM | null",
      "reviewed_at": "MM/DD/YYYY hh:mm AM/PM | null"
    }
  ],
  "total": 0,
  "page": 1,
  "limit": 25
}
```

**Error cases:**
- `401` — unauthorized
- `403` — insufficient role

---

### GET /api/candidates/{id}
Retrieve a single candidate profile in full detail.

**Required role:** INTAKE_SPECIALIST (own records), REVIEWER, ADMIN, AUDITOR

**Request headers:** `Authorization: Bearer <jwt>`

**Success 200:**
```json
{
  "id": "uuid",
  "first_name": "string",
  "last_name": "string",
  "date_of_birth": "string (masked if no VIEW_SENSITIVE)",
  "gender": "string",
  "contact_info": {},
  "exam_scores": {},
  "application_details": {},
  "transfer_preferences": {},
  "status": "string",
  "completeness_status": "string",
  "documents": [
    {
      "id": "uuid",
      "document_type": "string",
      "file_name": "string",
      "file_size": 0,
      "uploaded_at": "MM/DD/YYYY hh:mm AM/PM"
    }
  ],
  "checklist": [
    {
      "item_name": "string",
      "required": true,
      "fulfilled": false
    }
  ],
  "review_comment": "string | null",
  "created_at": "MM/DD/YYYY hh:mm AM/PM",
  "submitted_at": "MM/DD/YYYY hh:mm AM/PM | null",
  "reviewed_at": "MM/DD/YYYY hh:mm AM/PM | null"
}
```

**Error cases:**
- `401` — unauthorized
- `403` — insufficient role or attempting to access another specialist's record
- `404` — candidate not found

---

### PUT /api/candidates/{id}
Update a candidate profile. Only allowed when status is DRAFT.

**Required role:** INTAKE_SPECIALIST (own records), ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:** Same shape as POST /api/candidates (partial updates accepted; omitted fields unchanged)

**Success 200:**
```json
{
  "id": "uuid",
  "status": "DRAFT",
  "completeness_status": "INCOMPLETE | COMPLETE",
  "updated_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `400` — validation error
- `401` — unauthorized
- `403` — insufficient role, or candidate not in DRAFT status
- `404` — candidate not found

---

### POST /api/candidates/{id}/submit
Submit a candidate profile for review. Profile must have completeness_status COMPLETE.

**Required role:** INTAKE_SPECIALIST (own records), ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:** none

**Success 200:**
```json
{
  "id": "uuid",
  "status": "SUBMITTED",
  "submitted_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `400` — completeness_status is INCOMPLETE; response includes list of unfulfilled required checklist items
- `401` — unauthorized
- `403` — insufficient role
- `404` — candidate not found
- `409` — already submitted

---

### POST /api/candidates/{id}/approve
Approve a submitted candidate profile.

**Required role:** REVIEWER, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:**
```json
{
  "comment": "string (optional)"
}
```

**Success 200:**
```json
{
  "id": "uuid",
  "status": "APPROVED",
  "reviewed_by": "uuid",
  "reviewed_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `401` — unauthorized
- `403` — insufficient role
- `404` — candidate not found
- `409` — candidate not in SUBMITTED status

---

### POST /api/candidates/{id}/reject
Reject a submitted candidate profile. A comment is required.

**Required role:** REVIEWER, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:**
```json
{
  "comment": "string (required, must be non-empty)"
}
```

**Success 200:**
```json
{
  "id": "uuid",
  "status": "REJECTED",
  "review_comment": "string",
  "reviewed_by": "uuid",
  "reviewed_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `400` — comment is missing or empty
- `401` — unauthorized
- `403` — insufficient role
- `404` — candidate not found
- `409` — candidate not in SUBMITTED status

---

### POST /api/candidates/{id}/documents
Upload one or more documents and attach them to a candidate profile.

**Required role:** INTAKE_SPECIALIST (own records), ADMIN

**Request headers:** `Authorization: Bearer <jwt>`, `Content-Type: multipart/form-data`

**Request body:** multipart form data with fields:
- `document_type` (string, required per file)
- `file` (binary, required; PDF/JPG/PNG only, max 20 MB per file)

**Success 201:**
```json
{
  "documents": [
    {
      "id": "uuid",
      "document_type": "string",
      "file_name": "string",
      "file_size": 0,
      "sha256_hash": "string",
      "uploaded_at": "MM/DD/YYYY hh:mm AM/PM"
    }
  ]
}
```

**Error cases:**
- `400` — unsupported file type, file exceeds 20 MB, or missing document_type
- `401` — unauthorized
- `403` — insufficient role
- `404` — candidate not found
- `409` — candidate not in DRAFT status

---

## Lost & Found

### POST /api/listings
Create and publish a new lost-and-found listing.

**Required role:** INVENTORY_CLERK, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:**
```json
{
  "title": "string",
  "description": "string",
  "category_id": "uuid",
  "location_city": "string",
  "location_state": "string (2-letter US state code)",
  "location_description": "string (optional street/landmark detail)",
  "found_at": "ISO8601 datetime"
}
```

**Success 201:**
```json
{
  "id": "uuid",
  "status": "PUBLISHED",
  "duplicate_flag": false,
  "created_at": "MM/DD/YYYY hh:mm AM/PM"
}
```
If a duplicate is detected, `status` is `PENDING_REVIEW` and `duplicate_flag` is `true`, `duplicate_of_id` is populated.

**Error cases:**
- `400` — missing required fields, invalid state code, or invalid found_at
- `401` — unauthorized
- `403` — insufficient role

---

### GET /api/listings
List lost-and-found listings with filters.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:** `status`, `category_id`, `city`, `state`, `created_by`, `include_auto_unlisted` (boolean, default false), `page`, `limit`, `fuzzy` (boolean), `q` (search term)

**Success 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "title": "string",
      "category": { "id": "uuid", "name": "string" },
      "location_city": "string",
      "location_state": "string",
      "found_at": "MM/DD/YYYY hh:mm AM/PM",
      "status": "PUBLISHED | UNLISTED | DELETED | PENDING_REVIEW",
      "duplicate_flag": false,
      "auto_unlisted": false,
      "created_at": "MM/DD/YYYY hh:mm AM/PM"
    }
  ],
  "total": 0,
  "page": 1,
  "limit": 25
}
```

**Error cases:**
- `401` — unauthorized

---

### GET /api/listings/{id}
Retrieve a single listing in full detail.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Success 200:**
```json
{
  "id": "uuid",
  "title": "string",
  "description": "string",
  "category": { "id": "uuid", "name": "string" },
  "location_city": "string",
  "location_state": "string",
  "location_description": "string",
  "found_at": "MM/DD/YYYY hh:mm AM/PM",
  "status": "string",
  "duplicate_flag": false,
  "duplicate_of_id": "uuid | null",
  "auto_unlisted": false,
  "auto_unlisted_at": "MM/DD/YYYY hh:mm AM/PM | null",
  "override_by": "uuid | null",
  "override_at": "MM/DD/YYYY hh:mm AM/PM | null",
  "created_by": "uuid",
  "created_at": "MM/DD/YYYY hh:mm AM/PM",
  "updated_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `401` — unauthorized
- `404` — listing not found

---

### PUT /api/listings/{id}
Update a listing. Only allowed when status is PUBLISHED.

**Required role:** INVENTORY_CLERK (own listings), ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:** Same shape as POST /api/listings (partial updates accepted)

**Success 200:**
```json
{
  "id": "uuid",
  "updated_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `400` — validation error
- `401` — unauthorized
- `403` — insufficient role or not owner
- `404` — listing not found
- `409` — listing not in PUBLISHED status

---

### POST /api/listings/{id}/unlist
Manually unlist a published listing.

**Required role:** INVENTORY_CLERK (own listings), ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:** none

**Success 200:**
```json
{
  "id": "uuid",
  "status": "UNLISTED",
  "updated_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `401` — unauthorized
- `403` — insufficient role or not owner
- `404` — listing not found
- `409` — listing not in PUBLISHED status

---

### DELETE /api/listings/{id}
Permanently delete a listing. Soft-delete: status set to DELETED, row retained for audit.

**Required role:** INVENTORY_CLERK (own listings), ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Success 200:**
```json
{ "id": "uuid", "status": "DELETED" }
```

**Error cases:**
- `401` — unauthorized
- `403` — insufficient role or not owner
- `404` — listing not found

---

### POST /api/listings/{id}/override-duplicate
Override a duplicate flag and publish the listing. Requires a comment.

**Required role:** REVIEWER, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:**
```json
{
  "comment": "string (required)"
}
```

**Success 200:**
```json
{
  "id": "uuid",
  "status": "PUBLISHED",
  "duplicate_flag": true,
  "override_by": "uuid",
  "override_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `400` — comment is missing or empty
- `401` — unauthorized
- `403` — insufficient role
- `404` — listing not found
- `409` — listing not in PENDING_REVIEW status or duplicate_flag is false

---

## Automotive Parts

### POST /api/parts
Create a new part record with its initial version.

**Required role:** INVENTORY_CLERK, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:**
```json
{
  "part_number": "string (unique)",
  "name": "string",
  "description": "string",
  "fitment": {
    "model": "string",
    "year": "integer",
    "engine": "string",
    "transmission": "string"
  },
  "oem_number": "string",
  "alternative_numbers": ["string"],
  "attributes": {}
}
```

**Success 201:**
```json
{
  "id": "uuid",
  "part_number": "string",
  "active_version_id": "uuid",
  "version_number": 1,
  "created_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `400` — validation error or missing required fields
- `401` — unauthorized
- `403` — insufficient role
- `409` — part_number already exists

---

### GET /api/parts
List parts with optional filters.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:** `part_number`, `name`, `oem_number`, `model`, `year`, `page`, `limit`, `fuzzy` (boolean)

**Success 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "part_number": "string",
      "name": "string",
      "active_version": {
        "version_number": 1,
        "fitment": {},
        "oem_number": "string"
      },
      "created_at": "MM/DD/YYYY hh:mm AM/PM"
    }
  ],
  "total": 0,
  "page": 1,
  "limit": 25
}
```

**Error cases:**
- `401` — unauthorized

---

### GET /api/parts/{id}
Retrieve a single part with its active version details.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Success 200:**
```json
{
  "id": "uuid",
  "part_number": "string",
  "name": "string",
  "description": "string",
  "active_version": {
    "id": "uuid",
    "version_number": 1,
    "fitment": {
      "model": "string",
      "year": 0,
      "engine": "string",
      "transmission": "string"
    },
    "oem_number": "string",
    "alternative_numbers": ["string"],
    "attributes": {},
    "promoted_at": "MM/DD/YYYY hh:mm AM/PM | null",
    "created_at": "MM/DD/YYYY hh:mm AM/PM"
  },
  "created_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `401` — unauthorized
- `404` — part not found

---

### PUT /api/parts/{id}
Create a new pending version of a part with updated data.

**Required role:** INVENTORY_CLERK, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:** Same shape as POST /api/parts body (excluding part_number, which is immutable)

**Success 200:**
```json
{
  "id": "uuid",
  "new_version_id": "uuid",
  "version_number": 2,
  "note": "New version created. Call /promote to make it active."
}
```

**Error cases:**
- `400` — validation error
- `401` — unauthorized
- `403` — insufficient role
- `404` — part not found

---

### GET /api/parts/{id}/versions
Retrieve the full version history of a part.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:** `compare_a` (version_id), `compare_b` (version_id) — if both provided, response includes a diff object

**Success 200:**
```json
{
  "part_id": "uuid",
  "versions": [
    {
      "id": "uuid",
      "version_number": 1,
      "fitment": {},
      "oem_number": "string",
      "alternative_numbers": [],
      "attributes": {},
      "is_active": true,
      "promoted_by": "uuid | null",
      "promoted_at": "MM/DD/YYYY hh:mm AM/PM | null",
      "created_by": "uuid",
      "created_at": "MM/DD/YYYY hh:mm AM/PM"
    }
  ],
  "diff": null
}
```
When `compare_a` and `compare_b` are provided, `diff` contains a field-level comparison object showing changed, added, and removed fields between the two versions.

**Error cases:**
- `401` — unauthorized
- `404` — part not found

---

### POST /api/parts/{id}/promote
Promote a specific version to be the active version of a part.

**Required role:** INVENTORY_CLERK, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`

**Request body:**
```json
{
  "version_id": "uuid"
}
```

**Success 200:**
```json
{
  "part_id": "uuid",
  "active_version_id": "uuid",
  "version_number": 2,
  "promoted_at": "MM/DD/YYYY hh:mm AM/PM"
}
```

**Error cases:**
- `400` — version_id missing or does not belong to this part
- `401` — unauthorized
- `403` — insufficient role
- `404` — part or version not found
- `409` — version is already active

---

### POST /api/parts/import
Upload a CSV file for bulk part import. First call returns preview; second call (with `commit: true`) commits.

**Required role:** INVENTORY_CLERK, ADMIN

**Request headers:** `Authorization: Bearer <jwt>`, `Content-Type: multipart/form-data`

**Request body:** multipart form data:
- `file` (binary CSV, required)
- `commit` (boolean string "true"/"false", default "false")

**Success 200 (preview, commit=false):**
```json
{
  "job_id": "uuid",
  "status": "PREVIEW",
  "total_rows": 100,
  "valid_rows": 97,
  "error_rows": 3,
  "errors": [
    {
      "row": 12,
      "field": "year",
      "message": "must be a 4-digit integer"
    }
  ]
}
```

**Success 200 (commit=true, all rows valid):**
```json
{
  "job_id": "uuid",
  "status": "COMMITTED",
  "total_rows": 100,
  "valid_rows": 100,
  "error_rows": 0
}
```

**Error cases:**
- `400` — not a valid CSV, missing required columns
- `401` — unauthorized
- `403` — insufficient role
- `422` — commit attempted but validation errors exist (returns same error list); no data written

---

### GET /api/parts/export
Export parts catalog as CSV with configurable field selection.

**Required role:** INVENTORY_CLERK, ADMIN, AUDITOR (requires EXPORT permission)

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:** `fields` (comma-separated field names), `model`, `year`, `oem_number`

**Success 200:** CSV file download (`Content-Type: text/csv`, `Content-Disposition: attachment`)

**Error cases:**
- `401` — unauthorized
- `403` — missing EXPORT permission

---

## Search

### GET /api/search
Unified search across candidates, listings, and parts.

**Required role:** Any authenticated user

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:**
- `q` — search term (required)
- `types` — comma-separated entity types: `candidates,listings,parts` (default: all)
- `fuzzy` — boolean, enable fuzzy matching (default: false)
- `status` — filter by entity status
- `category_id` — filter listings by category
- `date_from`, `date_to` — ISO8601 date range filter
- `page`, `limit`

**Success 200:**
```json
{
  "results": {
    "candidates": [
      {
        "id": "uuid",
        "display_name": "string (masked if no VIEW_SENSITIVE)",
        "status": "string",
        "matched_fields": ["first_name"]
      }
    ],
    "listings": [
      {
        "id": "uuid",
        "title": "string",
        "category": "string",
        "status": "string",
        "matched_fields": ["title"]
      }
    ],
    "parts": [
      {
        "id": "uuid",
        "part_number": "string",
        "name": "string",
        "matched_fields": ["oem_number"]
      }
    ]
  },
  "total": 0,
  "page": 1,
  "limit": 25
}
```

**Error cases:**
- `400` — q parameter missing
- `401` — unauthorized

---

## Reports

### GET /api/reports/kpi
Return KPI metrics for the dashboard.

**Required role:** REVIEWER, ADMIN, AUDITOR

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:** `date_from`, `date_to` (ISO8601, required)

**Success 200:**
```json
{
  "period": {
    "from": "MM/DD/YYYY",
    "to": "MM/DD/YYYY"
  },
  "candidate_conversion_rate": {
    "submitted": 0,
    "approved": 0,
    "rate_percent": 0.0
  },
  "review_cycle_time": {
    "average_hours": 0.0,
    "median_hours": 0.0,
    "min_hours": 0.0,
    "max_hours": 0.0
  },
  "quota_utilization": [
    {
      "category": "string",
      "quota": 0,
      "used": 0,
      "utilization_percent": 0.0
    }
  ]
}
```

**Error cases:**
- `400` — missing or invalid date parameters
- `401` — unauthorized
- `403` — insufficient role

---

### GET /api/reports/export
Export a configurable report as CSV or JSON.

**Required role:** REVIEWER, ADMIN, AUDITOR (requires EXPORT permission)

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:**
- `type` — `candidates | listings | parts | audit` (required)
- `fields` — comma-separated field names for column selection
- `date_from`, `date_to` — ISO8601 date range
- `format` — `csv | json` (default: csv)
- `mask_sensitive` — boolean (default: true)

**Success 200:** File download (`Content-Disposition: attachment`)

**Error cases:**
- `400` — invalid type or format
- `401` — unauthorized
- `403` — missing EXPORT permission

---

## Audit Logs

### GET /api/audit-logs
Query the immutable audit log.

**Required role:** ADMIN, AUDITOR

**Request headers:** `Authorization: Bearer <jwt>`

**Query parameters:**
- `actor_id` — filter by user UUID
- `action` — filter by action type (e.g., `LOGIN`, `DOCUMENT_DOWNLOAD`, `CANDIDATE_APPROVED`)
- `target_type` — filter by entity type (e.g., `candidate`, `listing`, `part`)
- `target_id` — filter by entity UUID
- `date_from`, `date_to` — ISO8601 date range
- `page`, `limit`

**Success 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "actor_id": "uuid",
      "actor_username": "string",
      "actor_role": "string",
      "action": "string",
      "target_type": "string",
      "target_id": "uuid | null",
      "diff_json": {},
      "ip_address": "string",
      "device_id": "string",
      "occurred_at": "MM/DD/YYYY hh:mm AM/PM"
    }
  ],
  "total": 0,
  "page": 1,
  "limit": 25
}
```

**Error cases:**
- `401` — unauthorized
- `403` — insufficient role
