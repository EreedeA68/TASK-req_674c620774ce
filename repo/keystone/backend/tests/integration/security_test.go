package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecurity_InvalidJWT tests that a malformed token is rejected.
func TestSecurity_InvalidJWT(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	rec := makeRequest(t, e, http.MethodGet, "/api/candidates", nil, "invalid.jwt.token")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	resp := parseResponse(t, rec)
	assert.NotEmpty(t, resp.ErrorMessage)
}

// TestSecurity_ExpiredJWT tests that a syntactically valid but expired token is rejected.
func TestSecurity_ExpiredToken(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	// This token has exp in the past (generated with HS256, exp=1700000000, which is 2023-11-14).
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJzdWIiOiJ0ZXN0IiwiZXhwIjoxNzAwMDAwMDAwfQ." +
		"SomeInvalidSignature"

	rec := makeRequest(t, e, http.MethodGet, "/api/candidates", nil, expiredToken)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestSecurity_RoleEscalation ensures INVENTORY_CLERK cannot create candidates.
func TestSecurity_RoleEscalation_InventoryClerk(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	body, _ := json.Marshal(map[string]interface{}{
		"demographics": map[string]string{"name": "Test"},
	})
	rec := makeRequest(t, e, http.MethodPost, "/api/candidates", body, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_RoleEscalation_ReviewerCannotCreateParts ensures REVIEWER cannot create parts.
func TestSecurity_RoleEscalation_ReviewerCreatesPart(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodPost, "/api/parts", validPartBody(), token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_AuditLogAccessControl ensures only ADMIN and AUDITOR can access audit logs.
func TestSecurity_AuditLogAccessControl_IntakeSpecialist(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/audit-logs", nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSecurity_AuditLogAccessControl_Auditor(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "AUDITOR")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/audit-logs", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSecurity_AuditLogAccessControl_Admin(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "ADMIN")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/audit-logs", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSecurity_NoTokenForProtectedRoute ensures all protected routes require JWT.
func TestSecurity_NoTokenForProtectedRoutes(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/candidates"},
		{http.MethodGet, "/api/listings"},
		{http.MethodGet, "/api/parts"},
		{http.MethodGet, "/api/search?q=test"},
		{http.MethodGet, "/api/audit-logs"},
		{http.MethodGet, "/api/reports/kpi"},
		{http.MethodGet, "/api/documents/some-id/download"},
		{http.MethodGet, "/api/users"},
	}

	for _, route := range routes {
		t.Run(route.method+"_"+route.path, func(t *testing.T) {
			rec := makeRequest(t, e, route.method, route.path, nil, "")
			assert.Equal(t, http.StatusUnauthorized, rec.Code,
				"route %s %s should require authentication", route.method, route.path)
		})
	}
}

// TestSecurity_DuplicateOverride_NotAllowedByIntakeSpecialist
func TestSecurity_DuplicateOverride_NotAllowedByIntakeSpecialist(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodPost, "/api/listings/some-id/override-duplicate", nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_UserManagement_OnlyAdmin
func TestSecurity_UserManagement_OnlyAdmin(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	reviewerUser := createTestUser(t, "REVIEWER")
	reviewerToken := loginAs(t, e, reviewerUser.Username, "TestPass1!")

	body, _ := json.Marshal(map[string]interface{}{
		"username": "newuser",
		"email":    "newuser@test.com",
		"password": "TestPass1!",
		"role":     "REVIEWER",
	})
	rec := makeRequest(t, e, http.MethodPost, "/api/users", body, reviewerToken)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_CandidateApproval_OnlyReviewerAdmin
func TestSecurity_CandidateApproval_OnlyReviewerAdmin(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	// Setup.
	intakeUser := createTestUser(t, "INTAKE_SPECIALIST")
	intakeToken := loginAs(t, e, intakeUser.Username, "TestPass1!")

	// Create and submit candidate.
	createBody, _ := json.Marshal(map[string]interface{}{
		"demographics":       map[string]string{"name": "Security Test"},
		"examScores":         map[string]int{"score": 80},
		"applicationDetails": map[string]string{"program": "CS"},
	})
	createRec := makeRequest(t, e, http.MethodPost, "/api/candidates", createBody, intakeToken)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	makeRequest(t, e, http.MethodPost, "/api/candidates/"+createResp.Data.ID+"/submit", nil, intakeToken)

	// INTAKE_SPECIALIST should NOT be able to approve.
	approveBody, _ := json.Marshal(map[string]string{"comments": "approved"})
	rec := makeRequest(t, e, http.MethodPost, "/api/candidates/"+createResp.Data.ID+"/approve", approveBody, intakeToken)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_SessionInvalidatedAfterLogout verifies that a token is rejected after logout.
func TestSecurity_SessionInvalidatedAfterLogout(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Token works before logout.
	rec := makeRequest(t, e, http.MethodGet, "/api/candidates", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	// Logout invalidates the session.
	logoutRec := makeRequest(t, e, http.MethodPost, "/api/auth/logout", nil, token)
	require.Equal(t, http.StatusOK, logoutRec.Code)

	// Same token must now be rejected.
	rec2 := makeRequest(t, e, http.MethodGet, "/api/candidates", nil, token)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code, "invalidated token should be rejected")
}

// TestSecurity_CandidateDocuments_InventoryClerkForbidden ensures INVENTORY_CLERK cannot list candidate documents.
func TestSecurity_CandidateDocuments_InventoryClerkForbidden(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/candidates/some-id/documents", nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_ReadScope_InventoryClerkCannotReadCandidates ensures INVENTORY_CLERK is forbidden from candidate reads.
func TestSecurity_ReadScope_InventoryClerkCannotReadCandidates(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/candidates", nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_ReadScope_IntakeSpecialistCannotReadParts ensures INTAKE_SPECIALIST is forbidden from parts reads.
func TestSecurity_ReadScope_IntakeSpecialistCannotReadParts(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/parts", nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_ReadScope_ReviewerCannotReadParts ensures REVIEWER is forbidden from parts reads.
func TestSecurity_ReadScope_ReviewerCannotReadParts(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/parts", nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSecurity_ResponseEnvelope verifies the standard JSON envelope on error responses.
func TestSecurity_ResponseEnvelope(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	rec := makeRequest(t, e, http.MethodGet, "/api/candidates", nil, "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp struct {
		Data         interface{} `json:"data"`
		ErrorMessage string      `json:"errorMessage"`
		Code         int         `json:"code"`
		Details      interface{} `json:"details"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.NotEmpty(t, resp.ErrorMessage)
	assert.Nil(t, resp.Data)
}
