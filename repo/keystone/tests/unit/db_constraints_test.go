package unit_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// Database connection helper
// =============================================================================

// getTestDB opens a connection to the test PostgreSQL database using
// environment variables (or sensible defaults that match docker-compose.yml).
func getTestDB(t *testing.T) *sql.DB {
	t.Helper()

	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")
	name := envOrDefault("DB_NAME", "keystone")
	user := envOrDefault("DB_USER", "keystone")
	pass := envOrDefault("DB_PASSWORD", "keystone_pass")
	sslMode := envOrDefault("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		host, port, name, user, pass, sslMode,
	)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "sql.Open should not fail")

	// Verify connectivity
	require.NoError(t, db.Ping(), "database ping failed — is the DB running?")

	t.Cleanup(func() { db.Close() })
	return db
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// =============================================================================
// Constraint Tests — Unique
// =============================================================================

// TestUsersUniqueUsername verifies that inserting a duplicate username is
// rejected by the UNIQUE constraint on users.username.
func TestUsersUniqueUsername(t *testing.T) {
	db := getTestDB(t)

	// 'admin' already exists in seed data.
	_, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, role)
		VALUES ('admin', 'unique_email_for_test@example.com', 'hash', 'AUDITOR')
	`)
	require.Error(t, err, "expected unique constraint violation on username")
	assert.Contains(t, err.Error(), "unique", "error should mention unique constraint")
}

// TestUsersUniqueEmail verifies that inserting a duplicate email is rejected
// by the UNIQUE constraint on users.email.
func TestUsersUniqueEmail(t *testing.T) {
	db := getTestDB(t)

	// 'admin@keystone.local' already exists in seed data.
	_, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, role)
		VALUES ('unique_username_for_test', 'admin@keystone.local', 'hash', 'AUDITOR')
	`)
	require.Error(t, err, "expected unique constraint violation on email")
	assert.Contains(t, err.Error(), "unique", "error should mention unique constraint")
}

// =============================================================================
// Constraint Tests — Indexes
// =============================================================================

// TestCandidateDocumentsSHA256Index verifies that the index on
// candidate_documents(sha256_hash) exists in pg_indexes.
func TestCandidateDocumentsSHA256Index(t *testing.T) {
	db := getTestDB(t)

	var indexName string
	err := db.QueryRow(`
		SELECT indexname
		FROM   pg_indexes
		WHERE  tablename = 'candidate_documents'
		  AND  indexname = 'idx_candidate_documents_hash'
	`).Scan(&indexName)

	require.NoError(t, err, "index idx_candidate_documents_hash must exist")
	assert.Equal(t, "idx_candidate_documents_hash", indexName)
}

// TestListingsCategoryIndex verifies that the composite index on
// listings(category, created_at) exists in pg_indexes.
func TestListingsCategoryIndex(t *testing.T) {
	db := getTestDB(t)

	var indexName string
	err := db.QueryRow(`
		SELECT indexname
		FROM   pg_indexes
		WHERE  tablename = 'listings'
		  AND  indexname = 'idx_listings_category'
	`).Scan(&indexName)

	require.NoError(t, err, "index idx_listings_category must exist")
	assert.Equal(t, "idx_listings_category", indexName)
}

// =============================================================================
// Constraint Tests — Foreign Keys
// =============================================================================

// TestForeignKeyDocumentsCandidate verifies that inserting a candidate_document
// referencing a non-existent candidate_id raises a FK violation.
func TestForeignKeyDocumentsCandidate(t *testing.T) {
	db := getTestDB(t)

	nonExistentCandidateID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := db.Exec(`
		INSERT INTO candidate_documents (candidate_id, file_name, file_path, sha256_hash)
		VALUES ($1, 'test.pdf', '/tmp/test.pdf', 'deadbeefdeadbeefdeadbeefdeadbeef')
	`, nonExistentCandidateID)

	require.Error(t, err, "expected FK violation for non-existent candidate_id")
	assert.Contains(t, err.Error(), "foreign key", "error should mention foreign key constraint")
}

// TestForeignKeyPartVersions verifies that inserting a part_version referencing
// a non-existent part_id raises a FK violation.
func TestForeignKeyPartVersions(t *testing.T) {
	db := getTestDB(t)

	nonExistentPartID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := db.Exec(`
		INSERT INTO part_versions (part_id, version_number, change_summary)
		VALUES ($1, 1, 'orphan version')
	`, nonExistentPartID)

	require.Error(t, err, "expected FK violation for non-existent part_id")
	assert.Contains(t, err.Error(), "foreign key", "error should mention foreign key constraint")
}

// TestForeignKeyAuditLogs verifies that inserting an audit_log with a
// non-existent actor_id raises a FK violation.
func TestForeignKeyAuditLogs(t *testing.T) {
	db := getTestDB(t)

	nonExistentActorID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := db.Exec(`
		INSERT INTO audit_logs (actor_id, action, resource_type)
		VALUES ($1, 'TEST_ACTION', 'tests')
	`, nonExistentActorID)

	require.Error(t, err, "expected FK violation for non-existent actor_id")
	assert.Contains(t, err.Error(), "foreign key", "error should mention foreign key constraint")
}

// =============================================================================
// Constraint Tests — Immutability
// =============================================================================

// TestPartVersionImmutable verifies that the trigger fn_prevent_version_number_change
// prevents UPDATE of version_number on part_versions.
func TestPartVersionImmutable(t *testing.T) {
	db := getTestDB(t)

	// Seed a part and version to test immutability
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	var partID string
	err = tx.QueryRow(`
		INSERT INTO parts (part_number, name, status, created_by)
		VALUES ('KS-TEST-IMM-001', 'Immutability Test Part', 'DRAFT',
		        '00000000-0000-0000-0000-000000000004')
		RETURNING id
	`).Scan(&partID)
	require.NoError(t, err)

	var versionID string
	err = tx.QueryRow(`
		INSERT INTO part_versions (part_id, version_number, change_summary)
		VALUES ($1, 1, 'initial version for immutability test')
		RETURNING id
	`, partID).Scan(&versionID)
	require.NoError(t, err)

	// Attempt to change version_number — trigger should block this
	_, err = tx.Exec(`
		UPDATE part_versions SET version_number = 99 WHERE id = $1
	`, versionID)

	require.Error(t, err, "trigger should prevent version_number mutation")
	assert.Contains(t, err.Error(), "immutable", "trigger error message should say 'immutable'")
}

// =============================================================================
// Constraint Tests — CHECK Constraints / Enums
// =============================================================================

// TestListingsStatusEnum verifies that inserting a listing with an invalid
// status value is rejected by the CHECK constraint.
func TestListingsStatusEnum(t *testing.T) {
	db := getTestDB(t)

	_, err := db.Exec(`
		INSERT INTO listings (created_by, title, category, location_description, status)
		VALUES (
		    '00000000-0000-0000-0000-000000000001',
		    'Bad Status Listing',
		    'Test',
		    'Nowhere',
		    'INVALID'
		)
	`)

	require.Error(t, err, "expected CHECK constraint violation on listings.status")
	assert.Contains(t, err.Error(), "check", "error should mention check constraint")
}

// TestUsersRoleEnum verifies that inserting a user with an invalid role value
// is rejected by the CHECK constraint.
func TestUsersRoleEnum(t *testing.T) {
	db := getTestDB(t)

	_, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, role)
		VALUES ('roletest_user', 'roletest@example.com', 'hash', 'INVALID')
	`)

	require.Error(t, err, "expected CHECK constraint violation on users.role")
	assert.Contains(t, err.Error(), "check", "error should mention check constraint")
}

// =============================================================================
// Seed Data Tests
// =============================================================================

// seededUser holds data returned from the users query for verification.
type seededUser struct {
	username     string
	email        string
	role         string
	passwordHash string
}

// expectedSeedUsers defines all five users that init.sql must seed.
var expectedSeedUsers = []struct {
	username string
	email    string
	role     string
	password string
}{
	{"admin", "admin@keystone.local", "ADMIN", "Admin@Keystone1!"},
	{"intake_specialist", "intake@keystone.local", "INTAKE_SPECIALIST", "Intake@Keystone1!"},
	{"reviewer", "reviewer@keystone.local", "REVIEWER", "Review@Keystone1!"},
	{"inventory_clerk", "clerk@keystone.local", "INVENTORY_CLERK", "Clerk@Keystone1!"},
	{"auditor", "auditor@keystone.local", "AUDITOR", "Audit@Keystone1!"},
}

// TestSeedUsersExist verifies that all five seeded users are present in the
// database with the correct username, email, and role.
func TestSeedUsersExist(t *testing.T) {
	db := getTestDB(t)

	rows, err := db.Query(`
		SELECT username, email, role
		FROM   users
		WHERE  email IN (
		    'admin@keystone.local',
		    'intake@keystone.local',
		    'reviewer@keystone.local',
		    'clerk@keystone.local',
		    'auditor@keystone.local'
		)
		ORDER BY username
	`)
	require.NoError(t, err)
	defer rows.Close()

	found := make(map[string]struct{ email, role string })
	for rows.Next() {
		var uname, email, role string
		require.NoError(t, rows.Scan(&uname, &email, &role))
		found[uname] = struct{ email, role string }{email, role}
	}
	require.NoError(t, rows.Err())

	assert.Len(t, found, 5, "expected exactly 5 seeded users")

	for _, want := range expectedSeedUsers {
		got, ok := found[want.username]
		assert.True(t, ok, "user %q not found in DB", want.username)
		if ok {
			assert.Equal(t, want.email, got.email, "email mismatch for %s", want.username)
			assert.Equal(t, want.role, got.role, "role mismatch for %s", want.username)
		}
	}
}

// TestSeedPasswordsValid verifies that each seeded user's stored bcrypt hash
// correctly validates against the known plaintext password.
func TestSeedPasswordsValid(t *testing.T) {
	db := getTestDB(t)

	for _, want := range expectedSeedUsers {
		want := want // capture loop var
		t.Run(want.username, func(t *testing.T) {
			var hash string
			err := db.QueryRow(
				`SELECT password_hash FROM users WHERE email = $1`,
				want.email,
			).Scan(&hash)
			require.NoError(t, err, "user %q not found", want.username)
			require.NotEmpty(t, hash, "password_hash must not be empty")

			err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(want.password))
			assert.NoError(t, err,
				"bcrypt comparison failed for user %q (hash=%s)", want.username, hash)
		})
	}
}

// TestSeedCandidatesExist verifies that exactly 5 candidates were seeded.
func TestSeedCandidatesExist(t *testing.T) {
	db := getTestDB(t)

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM candidates`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 5, count, "expected exactly 5 seeded candidates")
}

// TestSeedCandidateStatuses verifies that candidates exist with the expected
// distribution of statuses as defined in the seed: DRAFT(2), SUBMITTED(1),
// APPROVED(1), REJECTED(1).
func TestSeedCandidateStatuses(t *testing.T) {
	db := getTestDB(t)

	rows, err := db.Query(`
		SELECT status, COUNT(*) AS cnt
		FROM   candidates
		GROUP  BY status
		ORDER  BY status
	`)
	require.NoError(t, err)
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var cnt int
		require.NoError(t, rows.Scan(&status, &cnt))
		statusCounts[status] = cnt
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, 2, statusCounts["DRAFT"], "expected 2 DRAFT candidates")
	assert.Equal(t, 1, statusCounts["SUBMITTED"], "expected 1 SUBMITTED candidate")
	assert.Equal(t, 1, statusCounts["APPROVED"], "expected 1 APPROVED candidate")
	assert.Equal(t, 1, statusCounts["REJECTED"], "expected 1 REJECTED candidate")
}

// TestSeedListingsExist verifies that exactly 10 listings were seeded, and
// that at least 2 have created_at older than 90 days (to test auto-unlist logic).
func TestSeedListingsExist(t *testing.T) {
	db := getTestDB(t)

	var totalCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM listings`).Scan(&totalCount)
	require.NoError(t, err)
	assert.Equal(t, 10, totalCount, "expected exactly 10 seeded listings")

	// Verify at least 2 listings are aged beyond 90 days
	ninetyDaysAgo := time.Now().UTC().Add(-90 * 24 * time.Hour)
	var oldCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM listings WHERE created_at < $1
	`, ninetyDaysAgo).Scan(&oldCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, oldCount, 2,
		"at least 2 listings must have created_at older than 90 days (got %d)", oldCount)
}

// TestSeedPartsExist verifies that exactly 10 parts and at least 20 part_versions
// (2 per part) were seeded.
func TestSeedPartsExist(t *testing.T) {
	db := getTestDB(t)

	var partCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM parts`).Scan(&partCount)
	require.NoError(t, err)
	assert.Equal(t, 10, partCount, "expected exactly 10 seeded parts")

	var versionCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM part_versions`).Scan(&versionCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, versionCount, 20,
		"expected at least 20 part_versions (2 per part), got %d", versionCount)
}

// TestSeedPartsCurrentVersionSet verifies that every seeded part has its
// current_version_id populated and points to a valid part_versions row.
func TestSeedPartsCurrentVersionSet(t *testing.T) {
	db := getTestDB(t)

	rows, err := db.Query(`
		SELECT p.id, p.part_number, p.current_version_id
		FROM   parts p
		WHERE  p.current_version_id IS NULL
	`)
	require.NoError(t, err)
	defer rows.Close()

	var nullVersionParts []string
	for rows.Next() {
		var id, partNumber string
		var currentVersionID *string
		require.NoError(t, rows.Scan(&id, &partNumber, &currentVersionID))
		nullVersionParts = append(nullVersionParts, partNumber)
	}
	require.NoError(t, rows.Err())

	assert.Empty(t, nullVersionParts,
		"parts with NULL current_version_id: %v", nullVersionParts)

	// Verify all current_version_ids reference existing part_versions
	var orphanCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM   parts p
		LEFT   JOIN part_versions pv ON pv.id = p.current_version_id
		WHERE  p.current_version_id IS NOT NULL
		  AND  pv.id IS NULL
	`).Scan(&orphanCount)
	require.NoError(t, err)
	assert.Equal(t, 0, orphanCount,
		"all current_version_id values must reference valid part_versions rows")
}

// TestSeedAuditLogsExist verifies that at least one audit log entry was seeded.
func TestSeedAuditLogsExist(t *testing.T) {
	db := getTestDB(t)

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM audit_logs`).Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1,
		"at least 1 audit_log entry must be seeded, got %d", count)
}

// TestSeedDownloadPermissionsExist verifies that at least one download permission
// was seeded.
func TestSeedDownloadPermissionsExist(t *testing.T) {
	db := getTestDB(t)

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM download_permissions`).Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1,
		"at least 1 download_permission entry must be seeded, got %d", count)
}

// =============================================================================
// Additional Index Tests
// =============================================================================

// TestAuditLogsActorIndex verifies the composite index on audit_logs(actor_id, created_at).
func TestAuditLogsActorIndex(t *testing.T) {
	db := getTestDB(t)

	var indexName string
	err := db.QueryRow(`
		SELECT indexname
		FROM   pg_indexes
		WHERE  tablename = 'audit_logs'
		  AND  indexname = 'idx_audit_logs_actor'
	`).Scan(&indexName)

	require.NoError(t, err, "index idx_audit_logs_actor must exist")
	assert.Equal(t, "idx_audit_logs_actor", indexName)
}

// TestAuditLogsResourceIndex verifies the composite index on
// audit_logs(resource_type, resource_id).
func TestAuditLogsResourceIndex(t *testing.T) {
	db := getTestDB(t)

	var indexName string
	err := db.QueryRow(`
		SELECT indexname
		FROM   pg_indexes
		WHERE  tablename = 'audit_logs'
		  AND  indexname = 'idx_audit_logs_resource'
	`).Scan(&indexName)

	require.NoError(t, err, "index idx_audit_logs_resource must exist")
	assert.Equal(t, "idx_audit_logs_resource", indexName)
}

// TestCandidateDocumentsCandidateIndex verifies the index on
// candidate_documents(candidate_id).
func TestCandidateDocumentsCandidateIndex(t *testing.T) {
	db := getTestDB(t)

	var indexName string
	err := db.QueryRow(`
		SELECT indexname
		FROM   pg_indexes
		WHERE  tablename = 'candidate_documents'
		  AND  indexname = 'idx_candidate_documents_candidate'
	`).Scan(&indexName)

	require.NoError(t, err, "index idx_candidate_documents_candidate must exist")
	assert.Equal(t, "idx_candidate_documents_candidate", indexName)
}

// TestPartVersionsPartIndex verifies the composite index on
// part_versions(part_id, version_number).
func TestPartVersionsPartIndex(t *testing.T) {
	db := getTestDB(t)

	var indexName string
	err := db.QueryRow(`
		SELECT indexname
		FROM   pg_indexes
		WHERE  tablename = 'part_versions'
		  AND  indexname = 'idx_part_versions_part'
	`).Scan(&indexName)

	require.NoError(t, err, "index idx_part_versions_part must exist")
	assert.Equal(t, "idx_part_versions_part", indexName)
}

// =============================================================================
// Extension Tests
// =============================================================================

// TestPgCryptoExtensionEnabled verifies that the pgcrypto extension is installed.
func TestPgCryptoExtensionEnabled(t *testing.T) {
	db := getTestDB(t)

	var extName string
	err := db.QueryRow(`
		SELECT extname FROM pg_extension WHERE extname = 'pgcrypto'
	`).Scan(&extName)

	require.NoError(t, err, "pgcrypto extension must be installed")
	assert.Equal(t, "pgcrypto", extName)
}

// =============================================================================
// Table Existence Tests
// =============================================================================

// TestAllTablesExist verifies that all required tables are present in the
// public schema.
func TestAllTablesExist(t *testing.T) {
	db := getTestDB(t)

	requiredTables := []string{
		"users",
		"sessions",
		"candidates",
		"candidate_documents",
		"listings",
		"parts",
		"part_versions",
		"part_fitments",
		"audit_logs",
		"download_permissions",
		"download_logs",
	}

	for _, tableName := range requiredTables {
		tableName := tableName // capture
		t.Run(tableName, func(t *testing.T) {
			var name string
			err := db.QueryRow(`
				SELECT table_name
				FROM   information_schema.tables
				WHERE  table_schema = 'public'
				  AND  table_name   = $1
			`, tableName).Scan(&name)
			require.NoError(t, err, "table %q must exist in schema 'public'", tableName)
			assert.Equal(t, tableName, name)
		})
	}
}

// =============================================================================
// Cascade Delete Tests
// =============================================================================

// TestCandidateDocumentsCascadeDelete verifies that deleting a candidate also
// deletes its associated documents (ON DELETE CASCADE).
func TestCandidateDocumentsCascadeDelete(t *testing.T) {
	db := getTestDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	// Create a temporary candidate
	var candidateID string
	err = tx.QueryRow(`
		INSERT INTO candidates (created_by, status)
		VALUES ('00000000-0000-0000-0000-000000000002', 'DRAFT')
		RETURNING id
	`).Scan(&candidateID)
	require.NoError(t, err)

	// Attach a document to it
	_, err = tx.Exec(`
		INSERT INTO candidate_documents (candidate_id, file_name, file_path, sha256_hash)
		VALUES ($1, 'cascade_test.pdf', '/tmp/cascade_test.pdf',
		        'cafecafecafecafecafecafecafecafecafecafecafecafecafecafecafecafe')
	`, candidateID)
	require.NoError(t, err)

	// Verify the document exists
	var docCount int
	err = tx.QueryRow(
		`SELECT COUNT(*) FROM candidate_documents WHERE candidate_id = $1`,
		candidateID,
	).Scan(&docCount)
	require.NoError(t, err)
	assert.Equal(t, 1, docCount, "document must exist before cascade test")

	// Delete the candidate
	_, err = tx.Exec(`DELETE FROM candidates WHERE id = $1`, candidateID)
	require.NoError(t, err)

	// Document must be gone
	err = tx.QueryRow(
		`SELECT COUNT(*) FROM candidate_documents WHERE candidate_id = $1`,
		candidateID,
	).Scan(&docCount)
	require.NoError(t, err)
	assert.Equal(t, 0, docCount, "documents must be cascade-deleted with candidate")
}

// =============================================================================
// Part Version Unique Constraint
// =============================================================================

// TestPartVersionUniqueConstraint verifies that the UNIQUE(part_id, version_number)
// constraint prevents duplicate version numbers for the same part.
func TestPartVersionUniqueConstraint(t *testing.T) {
	db := getTestDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	var partID string
	err = tx.QueryRow(`
		INSERT INTO parts (part_number, name, status, created_by)
		VALUES ('KS-TEST-UNIQ-002', 'Unique Version Test Part', 'DRAFT',
		        '00000000-0000-0000-0000-000000000004')
		RETURNING id
	`).Scan(&partID)
	require.NoError(t, err)

	_, err = tx.Exec(`
		INSERT INTO part_versions (part_id, version_number, change_summary)
		VALUES ($1, 1, 'first insert')
	`, partID)
	require.NoError(t, err)

	// Attempt to insert a second row with the same version_number
	_, err = tx.Exec(`
		INSERT INTO part_versions (part_id, version_number, change_summary)
		VALUES ($1, 1, 'duplicate version number')
	`, partID)

	require.Error(t, err, "expected unique constraint violation on (part_id, version_number)")
	assert.Contains(t, err.Error(), "unique", "error should mention unique constraint")
}

// =============================================================================
// Session table tests
// =============================================================================

// TestSessionForeignKeyUser verifies that a session cannot reference a
// non-existent user.
func TestSessionForeignKeyUser(t *testing.T) {
	db := getTestDB(t)

	nonExistentUserID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := db.Exec(`
		INSERT INTO sessions (user_id, token)
		VALUES ($1, 'some-random-token-value')
	`, nonExistentUserID)

	require.Error(t, err, "expected FK violation for non-existent user_id in sessions")
	assert.Contains(t, err.Error(), "foreign key", "error should mention foreign key constraint")
}
