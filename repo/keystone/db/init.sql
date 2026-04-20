-- =============================================================================
-- Keystone Phase 1 Database Initialization
-- PostgreSQL 15 Compatible
-- =============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- TABLES
-- =============================================================================

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username         VARCHAR(100) UNIQUE NOT NULL,
    email            VARCHAR(255) UNIQUE NOT NULL,
    password_hash    TEXT         NOT NULL,
    role             VARCHAR(30)  NOT NULL CHECK (role IN ('ADMIN','INTAKE_SPECIALIST','REVIEWER','INVENTORY_CLERK','AUDITOR')),
    is_locked        BOOLEAN      NOT NULL DEFAULT false,
    failed_attempts  INT          NOT NULL DEFAULT 0,
    lock_time        TIMESTAMP,
    mfa_secret       TEXT,
    mfa_enabled      BOOLEAN      NOT NULL DEFAULT false,
    site_id          VARCHAR(255),
    organization_id  VARCHAR(255),
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMP,
    last_active      TIMESTAMP,
    permissions      JSONB
);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id          UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT      UNIQUE NOT NULL,
    device_id   TEXT,
    ip_address  TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMP,
    invalidated BOOLEAN   NOT NULL DEFAULT false
);

-- Candidates table
CREATE TABLE IF NOT EXISTS candidates (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by           UUID        NOT NULL REFERENCES users(id),
    status               VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','SUBMITTED','APPROVED','REJECTED')),
    demographics         JSONB,
    exam_scores          JSONB,
    application_details  JSONB,
    transfer_preferences JSONB,
    completeness_status  VARCHAR(20) NOT NULL DEFAULT 'incomplete',
    site_id              VARCHAR(255),
    organization_id      VARCHAR(255),
    submitted_at         TIMESTAMP,
    reviewed_at          TIMESTAMP,
    reviewer_id          UUID        REFERENCES users(id),
    reviewer_comments    TEXT,
    created_at           TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP   NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMP
);

-- Candidate documents table
CREATE TABLE IF NOT EXISTS candidate_documents (
    id                UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id      UUID      NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    file_name         TEXT      NOT NULL,
    file_path         TEXT      NOT NULL,
    file_size         BIGINT,
    mime_type         TEXT,
    sha256_hash       TEXT      NOT NULL,
    uploader_id       UUID      REFERENCES users(id),
    watermark_enabled BOOLEAN   NOT NULL DEFAULT false,
    uploaded_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Listings table
CREATE TABLE IF NOT EXISTS listings (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by             UUID        NOT NULL REFERENCES users(id),
    title                  TEXT        NOT NULL,
    category               VARCHAR(100) NOT NULL,
    location_description   TEXT        NOT NULL,
    time_window_start      TIMESTAMP,
    time_window_end        TIMESTAMP,
    status                 VARCHAR(20) NOT NULL DEFAULT 'PUBLISHED' CHECK (status IN ('DRAFT','PUBLISHED','PENDING_REVIEW','UNLISTED','DELETED')),
    is_duplicate_flagged   BOOLEAN     NOT NULL DEFAULT false,
    site_id                VARCHAR(255),
    organization_id        VARCHAR(255),
    duplicate_override_by  UUID        REFERENCES users(id),
    duplicate_override_at  TIMESTAMP,
    unlisted_at            TIMESTAMP,
    unlisted_by            UUID        REFERENCES users(id),
    created_at             TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMP   NOT NULL DEFAULT NOW(),
    deleted_at             TIMESTAMP
);

-- Parts table
CREATE TABLE IF NOT EXISTS parts (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    part_number        VARCHAR(100) UNIQUE NOT NULL,
    name               TEXT         NOT NULL,
    description        TEXT,
    status             VARCHAR(20)  NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','DEPRECATED')),
    current_version_id UUID,
    site_id            VARCHAR(255),
    organization_id    VARCHAR(255),
    created_by         UUID         REFERENCES users(id),
    created_at         TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMP
);

-- Part versions table
CREATE TABLE IF NOT EXISTS part_versions (
    id             UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    part_id        UUID      NOT NULL REFERENCES parts(id) ON DELETE CASCADE,
    version_number INT       NOT NULL,
    fitment        JSONB,
    oem_mappings   JSONB,
    attributes     JSONB,
    changed_by     UUID      REFERENCES users(id),
    change_summary TEXT,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(part_id, version_number)
);

-- Add FK from parts to part_versions (deferred to avoid circular dependency)
ALTER TABLE parts ADD CONSTRAINT fk_parts_current_version
    FOREIGN KEY (current_version_id) REFERENCES part_versions(id)
    DEFERRABLE INITIALLY DEFERRED;

-- Part fitments table
CREATE TABLE IF NOT EXISTS part_fitments (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    part_id    UUID         NOT NULL REFERENCES parts(id) ON DELETE CASCADE,
    make       VARCHAR(100),
    model      VARCHAR(200),
    year_start INT,
    year_end   INT,
    engine     VARCHAR(100),
    created_at TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id              UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id        UUID      REFERENCES users(id),
    action          VARCHAR(100) NOT NULL,
    resource_type   VARCHAR(100),
    resource_id     UUID,
    before_state    JSONB,
    after_state     JSONB,
    device_id       TEXT,
    ip_address      TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Download permissions table
CREATE TABLE IF NOT EXISTS download_permissions (
    id            UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type TEXT,
    resource_id   UUID,
    granted_by    UUID      REFERENCES users(id),
    granted_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMP
);

-- Download logs table
CREATE TABLE IF NOT EXISTS download_logs (
    id                UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID      REFERENCES users(id),
    resource_type     TEXT,
    resource_id       UUID,
    downloaded_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    device_id         TEXT
);

-- =============================================================================
-- INDEXES
-- =============================================================================

CREATE INDEX idx_audit_logs_actor    ON audit_logs(actor_id, created_at);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_candidate_documents_candidate ON candidate_documents(candidate_id);
CREATE INDEX idx_candidate_documents_hash      ON candidate_documents(sha256_hash);
CREATE INDEX idx_listings_category             ON listings(category, created_at);
CREATE INDEX idx_part_versions_part            ON part_versions(part_id, version_number);

-- Additional useful indexes
CREATE INDEX idx_sessions_user      ON sessions(user_id);
CREATE INDEX idx_sessions_token     ON sessions(token);
CREATE INDEX idx_candidates_status  ON candidates(status);
CREATE INDEX idx_candidates_created_by ON candidates(created_by);
CREATE INDEX idx_listings_status    ON listings(status);
CREATE INDEX idx_download_permissions_user ON download_permissions(user_id, resource_id);
CREATE INDEX idx_users_org_scope       ON users(organization_id, site_id);
CREATE INDEX idx_candidates_org_scope  ON candidates(organization_id, site_id);
CREATE INDEX idx_listings_org_scope    ON listings(organization_id, site_id);
CREATE INDEX idx_parts_org_scope       ON parts(organization_id, site_id);
CREATE INDEX idx_candidates_deleted_at ON candidates(deleted_at);
CREATE INDEX idx_listings_deleted_at   ON listings(deleted_at);
CREATE INDEX idx_parts_deleted_at      ON parts(deleted_at);

-- =============================================================================
-- TRIGGERS
-- =============================================================================

-- Function to auto-update updated_at column
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_candidates_updated_at
    BEFORE UPDATE ON candidates
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_listings_updated_at
    BEFORE UPDATE ON listings
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_parts_updated_at
    BEFORE UPDATE ON parts
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- audit_logs rows are fully immutable: no updates or deletes allowed.
CREATE OR REPLACE FUNCTION fn_audit_logs_immutable()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'audit_logs rows are immutable and cannot be deleted';
    END IF;
    RAISE EXCEPTION 'audit_logs rows are immutable and cannot be updated';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_audit_logs_immutable
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION fn_audit_logs_immutable();

-- part_versions rows are fully immutable: no updates or deletes allowed.
CREATE OR REPLACE FUNCTION fn_part_versions_immutable()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'part_versions rows are immutable and cannot be deleted';
    END IF;
    RAISE EXCEPTION 'part_versions rows are immutable and cannot be updated';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_part_versions_immutable
    BEFORE UPDATE OR DELETE ON part_versions
    FOR EACH ROW EXECUTE FUNCTION fn_part_versions_immutable();

-- =============================================================================
-- SEED DATA
-- =============================================================================

-- -------------------------
-- Users (5 roles)
-- -------------------------
INSERT INTO users (id, username, email, password_hash, role, is_locked, failed_attempts, mfa_enabled, created_at)
VALUES
    (
        '00000000-0000-0000-0000-000000000001',
        'admin',
        'admin@keystone.local',
        crypt('Admin@Keystone1!', gen_salt('bf', 12)),
        'ADMIN',
        false, 0, false,
        NOW() - INTERVAL '180 days'
    ),
    (
        '00000000-0000-0000-0000-000000000002',
        'intake_specialist',
        'intake@keystone.local',
        crypt('Intake@Keystone1!', gen_salt('bf', 12)),
        'INTAKE_SPECIALIST',
        false, 0, false,
        NOW() - INTERVAL '150 days'
    ),
    (
        '00000000-0000-0000-0000-000000000003',
        'reviewer',
        'reviewer@keystone.local',
        crypt('Review@Keystone1!', gen_salt('bf', 12)),
        'REVIEWER',
        false, 0, false,
        NOW() - INTERVAL '120 days'
    ),
    (
        '00000000-0000-0000-0000-000000000004',
        'inventory_clerk',
        'clerk@keystone.local',
        crypt('Clerk@Keystone1!', gen_salt('bf', 12)),
        'INVENTORY_CLERK',
        false, 0, false,
        NOW() - INTERVAL '90 days'
    ),
    (
        '00000000-0000-0000-0000-000000000005',
        'auditor',
        'auditor@keystone.local',
        crypt('Audit@Keystone1!', gen_salt('bf', 12)),
        'AUDITOR',
        false, 0, false,
        NOW() - INTERVAL '60 days'
    );

-- -------------------------
-- Candidates (5 records, mixed statuses)
-- -------------------------
INSERT INTO candidates (id, created_by, status, demographics, exam_scores, application_details, transfer_preferences, completeness_status, submitted_at, reviewed_at, reviewer_id, reviewer_comments, created_at, updated_at)
VALUES
    (
        'c0000000-0000-0000-0000-000000000001',
        '00000000-0000-0000-0000-000000000002',
        'DRAFT',
        '{"first_name":"Alice","last_name":"Smith","dob":"1990-04-15","gender":"F","ethnicity":"Hispanic"}',
        '{"written":72,"practical":null,"oral":null}',
        '{"position":"Field Officer","department":"Operations","years_experience":3}',
        '{"preferred_locations":["North District","East District"],"max_commute_miles":30}',
        'incomplete',
        NULL,
        NULL,
        NULL,
        NULL,
        NOW() - INTERVAL '30 days',
        NOW() - INTERVAL '30 days'
    ),
    (
        'c0000000-0000-0000-0000-000000000002',
        '00000000-0000-0000-0000-000000000002',
        'SUBMITTED',
        '{"first_name":"Bob","last_name":"Johnson","dob":"1985-08-22","gender":"M","ethnicity":"Caucasian"}',
        '{"written":88,"practical":85,"oral":82}',
        '{"position":"Senior Analyst","department":"Intelligence","years_experience":7}',
        '{"preferred_locations":["Central HQ"],"max_commute_miles":15}',
        'complete',
        NOW() - INTERVAL '10 days',
        NULL,
        NULL,
        NULL,
        NOW() - INTERVAL '20 days',
        NOW() - INTERVAL '10 days'
    ),
    (
        'c0000000-0000-0000-0000-000000000003',
        '00000000-0000-0000-0000-000000000002',
        'APPROVED',
        '{"first_name":"Carol","last_name":"Williams","dob":"1992-12-01","gender":"F","ethnicity":"Asian"}',
        '{"written":95,"practical":91,"oral":93}',
        '{"position":"Investigator","department":"Compliance","years_experience":5}',
        '{"preferred_locations":["West District","Central HQ"],"max_commute_miles":25}',
        'complete',
        NOW() - INTERVAL '45 days',
        NOW() - INTERVAL '30 days',
        '00000000-0000-0000-0000-000000000003',
        'Outstanding scores and experience. Approved for immediate placement.',
        NOW() - INTERVAL '55 days',
        NOW() - INTERVAL '30 days'
    ),
    (
        'c0000000-0000-0000-0000-000000000004',
        '00000000-0000-0000-0000-000000000002',
        'REJECTED',
        '{"first_name":"David","last_name":"Brown","dob":"1998-03-10","gender":"M","ethnicity":"African American"}',
        '{"written":54,"practical":48,"oral":61}',
        '{"position":"Field Officer","department":"Operations","years_experience":1}',
        '{"preferred_locations":["South District"],"max_commute_miles":20}',
        'complete',
        NOW() - INTERVAL '25 days',
        NOW() - INTERVAL '15 days',
        '00000000-0000-0000-0000-000000000003',
        'Exam scores below minimum threshold. Recommend reapplication after additional training.',
        NOW() - INTERVAL '35 days',
        NOW() - INTERVAL '15 days'
    ),
    (
        'c0000000-0000-0000-0000-000000000005',
        '00000000-0000-0000-0000-000000000002',
        'DRAFT',
        '{"first_name":"Emma","last_name":"Davis","dob":"1995-07-18","gender":"F","ethnicity":"Caucasian"}',
        '{"written":null,"practical":null,"oral":null}',
        '{"position":"Administrative Coordinator","department":"Administration","years_experience":4}',
        '{"preferred_locations":["Central HQ","North District"],"max_commute_miles":10}',
        'incomplete',
        NULL,
        NULL,
        NULL,
        NULL,
        NOW() - INTERVAL '5 days',
        NOW() - INTERVAL '5 days'
    );

-- -------------------------
-- Candidate Documents
-- -------------------------
INSERT INTO candidate_documents (id, candidate_id, file_name, file_path, file_size, mime_type, sha256_hash, watermark_enabled, uploaded_at)
VALUES
    (
        'd0000000-0000-0000-0000-000000000001',
        'c0000000-0000-0000-0000-000000000002',
        'resume_bob_johnson.pdf',
        '/app/documents/candidates/c0000000-0000-0000-0000-000000000002/resume_bob_johnson.pdf',
        204800,
        'application/pdf',
        'a3f5c2e1b8d4f7a9e2c5b3d8f1a4c7e2b5d8f3a6c9e2b5d8f1a4c7e2b5d8f3a6',
        true,
        NOW() - INTERVAL '20 days'
    ),
    (
        'd0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000002',
        'transcript_bob_johnson.pdf',
        '/app/documents/candidates/c0000000-0000-0000-0000-000000000002/transcript_bob_johnson.pdf',
        153600,
        'application/pdf',
        'b4e6d3f2c9a7e4d1b6f3a8c5e2d9b4f7a2c5e8d1b6f3a8c5e2d9b4f7a2c5e8d1',
        true,
        NOW() - INTERVAL '19 days'
    ),
    (
        'd0000000-0000-0000-0000-000000000003',
        'c0000000-0000-0000-0000-000000000003',
        'resume_carol_williams.pdf',
        '/app/documents/candidates/c0000000-0000-0000-0000-000000000003/resume_carol_williams.pdf',
        307200,
        'application/pdf',
        'c5f7e4a3d0b8f5c2a9e6b3d0f7c4a1e8d5b2f9c6a3e0d7b4f1c8a5e2d9b6f3c0',
        true,
        NOW() - INTERVAL '55 days'
    ),
    (
        'd0000000-0000-0000-0000-000000000004',
        'c0000000-0000-0000-0000-000000000004',
        'resume_david_brown.pdf',
        '/app/documents/candidates/c0000000-0000-0000-0000-000000000004/resume_david_brown.pdf',
        102400,
        'application/pdf',
        'd6a8f5b4e1c9a6d3b0f7a4c1e8d5b2f9c6a3e0d7b4f1c8a5e2d9b6f3c0a7e4d1',
        false,
        NOW() - INTERVAL '35 days'
    );

-- -------------------------
-- Listings (10 records, 2 aged > 90 days)
-- -------------------------
INSERT INTO listings (id, created_by, title, category, location_description, time_window_start, time_window_end, status, is_duplicate_flagged, created_at, updated_at)
VALUES
    (
        'e0000000-0000-0000-0000-000000000001',
        '00000000-0000-0000-0000-000000000001',
        'AXLE SHAFT 2019 F-150 Front Left - OEM',
        'Drivetrain',
        'North Warehouse, Bay 3, Shelf A12',
        NOW() - INTERVAL '95 days' + INTERVAL '1 day',
        NOW() - INTERVAL '95 days' + INTERVAL '31 days',
        'PUBLISHED',
        false,
        NOW() - INTERVAL '95 days',
        NOW() - INTERVAL '95 days'
    ),
    (
        'e0000000-0000-0000-0000-000000000002',
        '00000000-0000-0000-0000-000000000001',
        'ALTERNATOR 2017 Honda Civic 1.5T - Remanufactured',
        'Electrical',
        'South Warehouse, Bay 7, Shelf C04',
        NOW() - INTERVAL '97 days' + INTERVAL '1 day',
        NOW() - INTERVAL '97 days' + INTERVAL '21 days',
        'PUBLISHED',
        false,
        NOW() - INTERVAL '97 days',
        NOW() - INTERVAL '97 days'
    ),
    (
        'e0000000-0000-0000-0000-000000000003',
        '00000000-0000-0000-0000-000000000004',
        'BRAKE CALIPER 2020 Camry Rear Right',
        'Brakes',
        'East Warehouse, Bay 2, Shelf B08',
        NOW() + INTERVAL '2 days',
        NOW() + INTERVAL '32 days',
        'PUBLISHED',
        false,
        NOW() - INTERVAL '5 days',
        NOW() - INTERVAL '5 days'
    ),
    (
        'e0000000-0000-0000-0000-000000000004',
        '00000000-0000-0000-0000-000000000004',
        'ENGINE BLOCK 2015 Silverado 5.3L - Core',
        'Engine',
        'Heavy Parts Yard, Section D, Slot 22',
        NOW() + INTERVAL '5 days',
        NOW() + INTERVAL '35 days',
        'DRAFT',
        false,
        NOW() - INTERVAL '3 days',
        NOW() - INTERVAL '3 days'
    ),
    (
        'e0000000-0000-0000-0000-000000000005',
        '00000000-0000-0000-0000-000000000004',
        'STRUT ASSEMBLY 2018 Accord Front Pair',
        'Suspension',
        'North Warehouse, Bay 5, Shelf D15',
        NOW() - INTERVAL '10 days',
        NOW() + INTERVAL '20 days',
        'PUBLISHED',
        false,
        NOW() - INTERVAL '10 days',
        NOW() - INTERVAL '10 days'
    ),
    (
        'e0000000-0000-0000-0000-000000000006',
        '00000000-0000-0000-0000-000000000004',
        'RADIATOR 2016 Jeep Grand Cherokee 3.6L',
        'Cooling',
        'South Warehouse, Bay 1, Shelf A01',
        NOW() - INTERVAL '5 days',
        NOW() + INTERVAL '25 days',
        'PUBLISHED',
        true,
        NOW() - INTERVAL '6 days',
        NOW() - INTERVAL '5 days'
    ),
    (
        'e0000000-0000-0000-0000-000000000007',
        '00000000-0000-0000-0000-000000000004',
        'TRANSMISSION 2014 Ram 1500 6-Speed Auto',
        'Transmission',
        'Heavy Parts Yard, Section B, Slot 08',
        NOW() + INTERVAL '1 day',
        NOW() + INTERVAL '31 days',
        'PUBLISHED',
        false,
        NOW() - INTERVAL '2 days',
        NOW() - INTERVAL '2 days'
    ),
    (
        'e0000000-0000-0000-0000-000000000008',
        '00000000-0000-0000-0000-000000000001',
        'POWER STEERING PUMP 2019 Tahoe',
        'Steering',
        'East Warehouse, Bay 4, Shelf E11',
        NOW() - INTERVAL '20 days',
        NOW() - INTERVAL '5 days',
        'UNLISTED',
        false,
        NOW() - INTERVAL '25 days',
        NOW() - INTERVAL '5 days'
    ),
    (
        'e0000000-0000-0000-0000-000000000009',
        '00000000-0000-0000-0000-000000000004',
        'FUEL INJECTOR SET 2017 Mustang GT 5.0L',
        'Fuel System',
        'North Warehouse, Bay 6, Shelf F03',
        NOW() + INTERVAL '3 days',
        NOW() + INTERVAL '33 days',
        'PUBLISHED',
        false,
        NOW() - INTERVAL '1 day',
        NOW() - INTERVAL '1 day'
    ),
    (
        'e0000000-0000-0000-0000-000000000010',
        '00000000-0000-0000-0000-000000000004',
        'CATALYTIC CONVERTER 2021 Prius - OEM',
        'Exhaust',
        'South Warehouse, Bay 9, Shelf G07',
        NOW() - INTERVAL '30 days',
        NOW() - INTERVAL '10 days',
        'DELETED',
        false,
        NOW() - INTERVAL '35 days',
        NOW() - INTERVAL '10 days'
    );

-- -------------------------
-- Parts (10 parts)
-- -------------------------
-- We insert parts first without current_version_id, then add versions, then update.

INSERT INTO parts (id, part_number, name, description, status, created_by, created_at, updated_at)
VALUES
    ('f0000000-0000-0000-0000-000000000001', 'KS-AX-001', 'Front Axle Shaft Assembly', 'Complete front axle shaft assembly for full-size trucks', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '200 days', NOW() - INTERVAL '30 days'),
    ('f0000000-0000-0000-0000-000000000002', 'KS-AL-001', 'Alternator 150A', '150-amp alternator for mid-size sedans', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '190 days', NOW() - INTERVAL '20 days'),
    ('f0000000-0000-0000-0000-000000000003', 'KS-BC-001', 'Brake Caliper Rear', 'Single-piston rear brake caliper', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '180 days', NOW() - INTERVAL '15 days'),
    ('f0000000-0000-0000-0000-000000000004', 'KS-EB-001', 'Engine Block V8 5.3L', 'Cast-iron V8 engine block', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '170 days', NOW() - INTERVAL '10 days'),
    ('f0000000-0000-0000-0000-000000000005', 'KS-SA-001', 'Strut Assembly Front', 'Complete strut assembly with spring and mount', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '160 days', NOW() - INTERVAL '25 days'),
    ('f0000000-0000-0000-0000-000000000006', 'KS-RA-001', 'Radiator Assembly', 'Aluminum core radiator with plastic tanks', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '150 days', NOW() - INTERVAL '5 days'),
    ('f0000000-0000-0000-0000-000000000007', 'KS-TR-001', 'Automatic Transmission 6-Speed', '6-speed automatic transmission assembly', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '140 days', NOW() - INTERVAL '45 days'),
    ('f0000000-0000-0000-0000-000000000008', 'KS-PS-001', 'Power Steering Pump', 'Hydraulic power steering pump', 'DEPRECATED', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '300 days', NOW() - INTERVAL '60 days'),
    ('f0000000-0000-0000-0000-000000000009', 'KS-FI-001', 'Fuel Injector 60lb/hr', 'High-flow fuel injector for performance engines', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '120 days', NOW() - INTERVAL '7 days'),
    ('f0000000-0000-0000-0000-000000000010', 'KS-CC-001', 'Catalytic Converter Rear', 'OEM-spec rear catalytic converter', 'ACTIVE', '00000000-0000-0000-0000-000000000004', NOW() - INTERVAL '110 days', NOW() - INTERVAL '3 days');

-- -------------------------
-- Part Versions (2+ per part = 20+)
-- -------------------------
INSERT INTO part_versions (id, part_id, version_number, fitment, oem_mappings, attributes, changed_by, change_summary, created_at)
VALUES
    -- Part 1: KS-AX-001 (Front Axle Shaft Assembly)
    ('b0000000-0000-0001-0001-000000000001', 'f0000000-0000-0000-0000-000000000001', 1,
     '{"makes":["Ford"],"models":["F-150"],"years":[2015,2016,2017,2018],"position":"Front Left"}',
     '{"ford_oem":"BL3Z-3B436-D","aftermarket":["Dorman 630-508"]}',
     '{"material":"4140 Steel","length_mm":572,"spline_count":28,"warranty_months":12}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '200 days'),
    ('b0000000-0000-0001-0001-000000000002', 'f0000000-0000-0000-0000-000000000001', 2,
     '{"makes":["Ford"],"models":["F-150","F-250"],"years":[2015,2016,2017,2018,2019],"position":"Front Left"}',
     '{"ford_oem":"BL3Z-3B436-D","ford_oem_v2":"FL3Z-3B436-A","aftermarket":["Dorman 630-508","GSP NCV12052"]}',
     '{"material":"4140 Steel","length_mm":572,"spline_count":28,"warranty_months":24,"notes":"Extended compatibility to F-250"}',
     '00000000-0000-0000-0000-000000000004', 'Added F-250 compatibility and extended warranty', NOW() - INTERVAL '30 days'),

    -- Part 2: KS-AL-001 (Alternator 150A)
    ('b0000000-0000-0001-0002-000000000001', 'f0000000-0000-0000-0000-000000000002', 1,
     '{"makes":["Honda"],"models":["Civic"],"years":[2016,2017,2018],"engine":"1.5T"}',
     '{"honda_oem":"31100-59B-014","aftermarket":["Remy 94793","DB Electrical AND0544"]}',
     '{"output_amps":150,"voltage":12,"rotation":"CW","warranty_months":18}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '190 days'),
    ('b0000000-0000-0001-0002-000000000002', 'f0000000-0000-0000-0000-000000000002', 2,
     '{"makes":["Honda"],"models":["Civic","Accord"],"years":[2016,2017,2018,2019],"engine":"1.5T"}',
     '{"honda_oem":"31100-59B-014","honda_oem_v2":"31100-5PA-A01","aftermarket":["Remy 94793","DB Electrical AND0544","WAI 11394N"]}',
     '{"output_amps":150,"voltage":12,"rotation":"CW","warranty_months":24,"notes":"Added Accord 1.5T fitment"}',
     '00000000-0000-0000-0000-000000000004', 'Extended fitment to Accord 1.5T', NOW() - INTERVAL '20 days'),

    -- Part 3: KS-BC-001 (Brake Caliper Rear)
    ('b0000000-0000-0001-0003-000000000001', 'f0000000-0000-0000-0000-000000000003', 1,
     '{"makes":["Toyota"],"models":["Camry"],"years":[2018,2019,2020],"position":"Rear Right"}',
     '{"toyota_oem":"47750-06190","aftermarket":["Cardone 19-3595","PowerStop S4946"]}',
     '{"piston_count":1,"piston_diameter_mm":38,"material":"Cast Iron","warranty_months":12}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '180 days'),
    ('b0000000-0000-0001-0003-000000000002', 'f0000000-0000-0000-0000-000000000003', 2,
     '{"makes":["Toyota"],"models":["Camry","Avalon"],"years":[2018,2019,2020,2021],"position":"Rear Right"}',
     '{"toyota_oem":"47750-06190","aftermarket":["Cardone 19-3595","PowerStop S4946","Centric 141.44592"]}',
     '{"piston_count":1,"piston_diameter_mm":38,"material":"Cast Iron","warranty_months":24,"coating":"E-coat"}',
     '00000000-0000-0000-0000-000000000004', 'Added Avalon fitment and E-coat finish', NOW() - INTERVAL '15 days'),

    -- Part 4: KS-EB-001 (Engine Block V8 5.3L)
    ('b0000000-0000-0001-0004-000000000001', 'f0000000-0000-0000-0000-000000000004', 1,
     '{"makes":["Chevrolet","GMC"],"models":["Silverado 1500","Sierra 1500"],"years":[2014,2015,2016],"engine":"5.3L V8 L83"}',
     '{"gm_oem":"12681429","aftermarket":["Dart 31364265"]}',
     '{"displacement_cc":5328,"bore_mm":96.52,"stroke_mm":92.0,"material":"Cast Iron","warranty_months":6}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '170 days'),
    ('b0000000-0000-0001-0004-000000000002', 'f0000000-0000-0000-0000-000000000004', 2,
     '{"makes":["Chevrolet","GMC","Cadillac"],"models":["Silverado 1500","Sierra 1500","Escalade"],"years":[2014,2015,2016,2017],"engine":"5.3L V8 L83"}',
     '{"gm_oem":"12681429","gm_oem_v2":"12690717","aftermarket":["Dart 31364265"]}',
     '{"displacement_cc":5328,"bore_mm":96.52,"stroke_mm":92.0,"material":"Cast Iron","warranty_months":6,"notes":"Added Escalade and 2017MY"}',
     '00000000-0000-0000-0000-000000000004', 'Expanded fitment to Cadillac Escalade', NOW() - INTERVAL '10 days'),

    -- Part 5: KS-SA-001 (Strut Assembly Front)
    ('b0000000-0000-0001-0005-000000000001', 'f0000000-0000-0000-0000-000000000005', 1,
     '{"makes":["Honda"],"models":["Accord"],"years":[2016,2017,2018],"position":"Front Left"}',
     '{"honda_oem":"51601-TVA-A04","aftermarket":["KYB SR4518","Monroe 172584"]}',
     '{"spring_rate_nm":28000,"travel_mm":110,"warranty_months":12}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '160 days'),
    ('b0000000-0000-0001-0005-000000000002', 'f0000000-0000-0000-0000-000000000005', 2,
     '{"makes":["Honda"],"models":["Accord","Acura TLX"],"years":[2016,2017,2018,2019],"position":"Front Left"}',
     '{"honda_oem":"51601-TVA-A04","aftermarket":["KYB SR4518","Monroe 172584","Gabriel G57131"]}',
     '{"spring_rate_nm":28000,"travel_mm":110,"warranty_months":24,"notes":"Added Acura TLX cross-reference"}',
     '00000000-0000-0000-0000-000000000004', 'Added Acura TLX compatibility', NOW() - INTERVAL '25 days'),

    -- Part 6: KS-RA-001 (Radiator Assembly)
    ('b0000000-0000-0001-0006-000000000001', 'f0000000-0000-0000-0000-000000000006', 1,
     '{"makes":["Jeep"],"models":["Grand Cherokee"],"years":[2014,2015,2016],"engine":"3.6L V6"}',
     '{"mopar_oem":"68110299AA","aftermarket":["Spectra Premium CU13384","TYC 13384"]}',
     '{"core_width_mm":680,"core_height_mm":400,"row_count":2,"material":"Aluminum","warranty_months":12}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '150 days'),
    ('b0000000-0000-0001-0006-000000000002', 'f0000000-0000-0000-0000-000000000006', 2,
     '{"makes":["Jeep","Dodge"],"models":["Grand Cherokee","Durango"],"years":[2014,2015,2016,2017],"engine":"3.6L V6"}',
     '{"mopar_oem":"68110299AA","mopar_oem_v2":"68303655AA","aftermarket":["Spectra Premium CU13384","TYC 13384","CSF 3561"]}',
     '{"core_width_mm":680,"core_height_mm":400,"row_count":2,"material":"Aluminum","warranty_months":18,"notes":"Added Dodge Durango fitment"}',
     '00000000-0000-0000-0000-000000000004', 'Added Durango compatibility and improved warranty', NOW() - INTERVAL '5 days'),

    -- Part 7: KS-TR-001 (Automatic Transmission 6-Speed)
    ('b0000000-0000-0001-0007-000000000001', 'f0000000-0000-0000-0000-000000000007', 1,
     '{"makes":["Ram"],"models":["1500"],"years":[2013,2014,2015],"engine":"5.7L Hemi"}',
     '{"mopar_oem":"5170903AF","aftermarket":["A-1 Cardone 77-4559"]}',
     '{"gear_count":6,"torque_rating_nm":700,"fluid_type":"ATF+4","warranty_months":6}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '140 days'),
    ('b0000000-0000-0001-0007-000000000002', 'f0000000-0000-0000-0000-000000000007', 2,
     '{"makes":["Ram","Dodge"],"models":["1500","Challenger"],"years":[2013,2014,2015,2016],"engine":"5.7L Hemi"}',
     '{"mopar_oem":"5170903AF","mopar_oem_v2":"68271639AA","aftermarket":["A-1 Cardone 77-4559"]}',
     '{"gear_count":6,"torque_rating_nm":700,"fluid_type":"ATF+4","warranty_months":6,"notes":"Added Challenger Hemi fitment"}',
     '00000000-0000-0000-0000-000000000004', 'Extended fitment to Dodge Challenger Hemi', NOW() - INTERVAL '45 days'),

    -- Part 8: KS-PS-001 (Power Steering Pump) - DEPRECATED
    ('b0000000-0000-0001-0008-000000000001', 'f0000000-0000-0000-0000-000000000008', 1,
     '{"makes":["Chevrolet"],"models":["Tahoe","Suburban"],"years":[2015,2016,2017,2018,2019]}',
     '{"gm_oem":"84145526","aftermarket":["Cardone 20-5224","ACDelco 84145526"]}',
     '{"flow_rate_lpm":8.5,"pressure_bar":130,"warranty_months":12}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '300 days'),
    ('b0000000-0000-0001-0008-000000000002', 'f0000000-0000-0000-0000-000000000008', 2,
     '{"makes":["Chevrolet","GMC"],"models":["Tahoe","Suburban","Yukon","Yukon XL"],"years":[2015,2016,2017,2018,2019]}',
     '{"gm_oem":"84145526","aftermarket":["Cardone 20-5224","ACDelco 84145526","Maval 96637M"]}',
     '{"flow_rate_lpm":8.5,"pressure_bar":130,"warranty_months":12,"notes":"Added GMC Yukon family; deprecated due to EPS conversion"}',
     '00000000-0000-0000-0000-000000000004', 'Added GMC Yukon; part marked deprecated (EPS supersedes)', NOW() - INTERVAL '60 days'),

    -- Part 9: KS-FI-001 (Fuel Injector 60lb/hr)
    ('b0000000-0000-0001-0009-000000000001', 'f0000000-0000-0000-0000-000000000009', 1,
     '{"makes":["Ford"],"models":["Mustang"],"years":[2015,2016,2017],"engine":"5.0L V8 Coyote"}',
     '{"ford_oem":"BR3E-9F593-AA","aftermarket":["Deatschwerks 18U-00-0060-8","Injector Dynamics ID1050x"]}',
     '{"flow_rate_lbhr":60,"impedance_ohm":12,"connector":"EV6 USCAR","warranty_months":12}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '120 days'),
    ('b0000000-0000-0001-0009-000000000002', 'f0000000-0000-0000-0000-000000000009', 2,
     '{"makes":["Ford"],"models":["Mustang","F-150"],"years":[2015,2016,2017,2018,2019],"engine":"5.0L V8 Coyote"}',
     '{"ford_oem":"BR3E-9F593-AA","aftermarket":["Deatschwerks 18U-00-0060-8","Injector Dynamics ID1050x","Siemens Deka FI114992"]}',
     '{"flow_rate_lbhr":60,"impedance_ohm":12,"connector":"EV6 USCAR","warranty_months":24,"notes":"Added F-150 5.0L fitment; extended warranty"}',
     '00000000-0000-0000-0000-000000000004', 'Added F-150 5.0L fitment and extended warranty', NOW() - INTERVAL '7 days'),

    -- Part 10: KS-CC-001 (Catalytic Converter Rear)
    ('b0000000-0000-0001-0010-000000000001', 'f0000000-0000-0000-0000-000000000010', 1,
     '{"makes":["Toyota"],"models":["Prius","Prius Prime"],"years":[2016,2017,2018,2019,2020]}',
     '{"toyota_oem":"25051-37250","aftermarket":["Eastern 30740","Davico 19378"]}',
     '{"substrate":"Palladium-Rhodium","cell_count":400,"inlet_diameter_mm":52,"warranty_months":60,"notes":"Federal emissions compliant"}',
     '00000000-0000-0000-0000-000000000004', 'Initial release', NOW() - INTERVAL '110 days'),
    ('b0000000-0000-0001-0010-000000000002', 'f0000000-0000-0000-0000-000000000010', 2,
     '{"makes":["Toyota"],"models":["Prius","Prius Prime","Prius AWD-e"],"years":[2016,2017,2018,2019,2020,2021]}',
     '{"toyota_oem":"25051-37250","toyota_oem_v2":"25051-47110","aftermarket":["Eastern 30740","Davico 19378","MagnaFlow 5481609"]}',
     '{"substrate":"Palladium-Rhodium","cell_count":400,"inlet_diameter_mm":52,"warranty_months":60,"notes":"Added AWD-e variant and 2021MY; CARB compliant"}',
     '00000000-0000-0000-0000-000000000004', 'Added Prius AWD-e and CARB compliance', NOW() - INTERVAL '3 days');

-- -------------------------
-- Update parts.current_version_id to latest version
-- -------------------------
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0001-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000001';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0002-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000002';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0003-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000003';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0004-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000004';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0005-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000005';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0006-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000006';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0007-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000007';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0008-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000008';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0009-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000009';
UPDATE parts SET current_version_id = 'b0000000-0000-0001-0010-000000000002' WHERE id = 'f0000000-0000-0000-0000-000000000010';

-- -------------------------
-- Part Fitment records
-- -------------------------
INSERT INTO part_fitments (id, part_id, make, model, year_start, year_end, engine)
VALUES
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000001', 'Ford', 'F-150', 2018, 2019, '5.0L V8'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000001', 'Ford', 'F-150', 2018, 2019, '3.5L EcoBoost'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000002', 'Honda', 'Civic', 2016, 2019, '1.5T'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000002', 'Honda', 'Accord', 2016, 2019, '1.5T'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000003', 'Toyota', 'Camry', 2018, 2021, '2.5L 4-Cylinder'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000004', 'Chevrolet', 'Silverado 1500', 2014, 2017, '5.3L V8 L83'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000005', 'Honda', 'Accord', 2016, 2019, '2.0T'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000006', 'Jeep', 'Grand Cherokee', 2014, 2017, '3.6L V6'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000007', 'Ram', '1500', 2013, 2016, '5.7L Hemi'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000009', 'Ford', 'Mustang', 2015, 2019, '5.0L V8 Coyote'),
    (gen_random_uuid(), 'f0000000-0000-0000-0000-000000000010', 'Toyota', 'Prius', 2016, 2021, '1.8L Hybrid');

-- -------------------------
-- Audit Log Entries
-- -------------------------
INSERT INTO audit_logs (id, actor_id, action, resource_type, resource_id, before_state, after_state, device_id, ip_address, created_at)
VALUES
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000001',
        'USER_CREATED',
        'users',
        '00000000-0000-0000-0000-000000000002',
        NULL,
        '{"username":"intake_specialist","email":"intake@keystone.local","role":"INTAKE_SPECIALIST"}',
        'server-init',
        '127.0.0.1',
        NOW() - INTERVAL '180 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000002',
        'CANDIDATE_SUBMITTED',
        'candidates',
        'c0000000-0000-0000-0000-000000000002',
        '{"status":"DRAFT"}',
        ('{"status":"SUBMITTED","submitted_at":"' || (NOW() - INTERVAL '10 days')::TEXT || '"}')::jsonb,
        'chrome-win-001',
        '192.168.1.105',
        NOW() - INTERVAL '10 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000003',
        'CANDIDATE_APPROVED',
        'candidates',
        'c0000000-0000-0000-0000-000000000003',
        '{"status":"SUBMITTED"}',
        ('{"status":"APPROVED","reviewer_id":"00000000-0000-0000-0000-000000000003","reviewed_at":"' || (NOW() - INTERVAL '30 days')::TEXT || '"}')::jsonb,
        'firefox-mac-007',
        '192.168.1.210',
        NOW() - INTERVAL '30 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000003',
        'CANDIDATE_REJECTED',
        'candidates',
        'c0000000-0000-0000-0000-000000000004',
        '{"status":"SUBMITTED"}',
        ('{"status":"REJECTED","reviewer_id":"00000000-0000-0000-0000-000000000003","reviewed_at":"' || (NOW() - INTERVAL '15 days')::TEXT || '"}')::jsonb,
        'firefox-mac-007',
        '192.168.1.210',
        NOW() - INTERVAL '15 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000004',
        'LISTING_CREATED',
        'listings',
        'e0000000-0000-0000-0000-000000000003',
        NULL,
        '{"title":"BRAKE CALIPER 2020 Camry Rear Right","category":"Brakes","status":"PUBLISHED"}',
        'edge-win-003',
        '10.0.0.55',
        NOW() - INTERVAL '5 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000004',
        'PART_CREATED',
        'parts',
        'f0000000-0000-0000-0000-000000000001',
        NULL,
        '{"part_number":"KS-AX-001","name":"Front Axle Shaft Assembly","status":"ACTIVE"}',
        'edge-win-003',
        '10.0.0.55',
        NOW() - INTERVAL '200 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000004',
        'PART_VERSION_ADDED',
        'part_versions',
        'b0000000-0000-0001-0001-000000000002',
        '{"version_number":1,"fitment":{"makes":["Ford"],"models":["F-150"]}}',
        '{"version_number":2,"fitment":{"makes":["Ford"],"models":["F-150","F-250"]}}',
        'edge-win-003',
        '10.0.0.55',
        NOW() - INTERVAL '30 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000001',
        'LISTING_DUPLICATE_FLAGGED',
        'listings',
        'e0000000-0000-0000-0000-000000000006',
        '{"is_duplicate_flagged":false}',
        '{"is_duplicate_flagged":true}',
        'chrome-win-001',
        '192.168.1.1',
        NOW() - INTERVAL '5 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000005',
        'AUDIT_REPORT_GENERATED',
        'audit_logs',
        NULL,
        NULL,
        '{"report_type":"monthly","period":"2026-03","record_count":147}',
        'safari-mac-012',
        '192.168.1.99',
        NOW() - INTERVAL '19 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000002',
        'DOCUMENT_UPLOADED',
        'candidate_documents',
        'd0000000-0000-0000-0000-000000000001',
        NULL,
        '{"file_name":"resume_bob_johnson.pdf","candidate_id":"c0000000-0000-0000-0000-000000000002","sha256_hash":"a3f5c2e1b8d4f7a9e2c5b3d8f1a4c7e2b5d8f3a6c9e2b5d8f1a4c7e2b5d8f3a6"}',
        'chrome-win-001',
        '192.168.1.105',
        NOW() - INTERVAL '20 days'
    );

-- -------------------------
-- Download Permissions
-- -------------------------
INSERT INTO download_permissions (id, user_id, resource_type, resource_id, granted_by, granted_at, expires_at)
VALUES
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000003',
        'candidate_document',
        'd0000000-0000-0000-0000-000000000001',
        '00000000-0000-0000-0000-000000000001',
        NOW() - INTERVAL '10 days',
        NOW() + INTERVAL '20 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000003',
        'candidate_document',
        'd0000000-0000-0000-0000-000000000002',
        '00000000-0000-0000-0000-000000000001',
        NOW() - INTERVAL '10 days',
        NOW() + INTERVAL '20 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000003',
        'candidate_document',
        'd0000000-0000-0000-0000-000000000003',
        '00000000-0000-0000-0000-000000000001',
        NOW() - INTERVAL '30 days',
        NOW() + INTERVAL '30 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000005',
        'candidate_document',
        'd0000000-0000-0000-0000-000000000004',
        '00000000-0000-0000-0000-000000000001',
        NOW() - INTERVAL '15 days',
        NOW() + INTERVAL '75 days'
    ),
    (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000002',
        'listings',
        'e0000000-0000-0000-0000-000000000001',
        '00000000-0000-0000-0000-000000000001',
        NOW() - INTERVAL '90 days',
        NOW() - INTERVAL '1 day'
    );

-- -------------------------
-- Download Logs
-- -------------------------
INSERT INTO download_logs (id, user_id, resource_type, resource_id, downloaded_at, device_id)
VALUES
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000003', 'candidate_document', 'd0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '9 days', 'firefox-mac-007'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000003', 'candidate_document', 'd0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '9 days', 'firefox-mac-007'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000003', 'candidate_document', 'd0000000-0000-0000-0000-000000000003', NOW() - INTERVAL '28 days', 'firefox-mac-007'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000005', 'audit_logs',           NULL,                                   NOW() - INTERVAL '19 days', 'safari-mac-012'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000002', 'listings',              'e0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '88 days', 'chrome-win-001');
