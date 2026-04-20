package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCandidateCreate_AsIntakeSpecialist(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")

	body, _ := json.Marshal(map[string]interface{}{
		"demographics":        map[string]string{"name": "John Doe", "dob": "1990-01-01"},
		"examScores":          map[string]int{"math": 90, "english": 85},
		"applicationDetails":  map[string]string{"program": "CS"},
		"transferPreferences": map[string]string{"city": "Austin"},
	})

	rec := makeRequest(t, e, http.MethodPost, "/api/candidates", body, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp struct {
		Data struct {
			ID                 string `json:"id"`
			Status             string `json:"status"`
			CompletenessStatus string `json:"completenessStatus"`
		} `json:"data"`
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.ID)
	assert.Equal(t, "DRAFT", resp.Data.Status)
	assert.Equal(t, "complete", resp.Data.CompletenessStatus)
	assert.Equal(t, 201, resp.Code)
}

func TestCandidateCreate_AsReviewer_Forbidden(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")

	body, _ := json.Marshal(map[string]interface{}{
		"demographics": map[string]string{"name": "Jane Doe"},
	})

	rec := makeRequest(t, e, http.MethodPost, "/api/candidates", body, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCandidateCreate_Incomplete(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Only demographics, missing exam scores and application details.
	body, _ := json.Marshal(map[string]interface{}{
		"demographics": map[string]string{"name": "John Doe"},
	})

	rec := makeRequest(t, e, http.MethodPost, "/api/candidates", body, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp struct {
		Data struct {
			CompletenessStatus string `json:"completenessStatus"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "incomplete", resp.Data.CompletenessStatus)
}

func TestCandidateList(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/candidates", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Items []interface{} `json:"items"`
			Total int64         `json:"total"`
			Page  int           `json:"page"`
			Limit int           `json:"limit"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data.Items)
	assert.GreaterOrEqual(t, resp.Data.Total, int64(0))
}

func TestCandidateGet_NotFound(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/candidates/00000000-0000-0000-0000-000000000000", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCandidateSubmit_Incomplete(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Create incomplete draft.
	createBody, _ := json.Marshal(map[string]interface{}{
		"demographics": map[string]string{"name": "John"},
	})
	createRec := makeRequest(t, e, http.MethodPost, "/api/candidates", createBody, token)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))

	// Try to submit – should fail because it's incomplete.
	rec := makeRequest(t, e, http.MethodPost, "/api/candidates/"+createResp.Data.ID+"/submit", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCandidateApprove_ByReviewer(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	intakeUser := createTestUser(t, "INTAKE_SPECIALIST")
	intakeToken := loginAs(t, e, intakeUser.Username, "TestPass1!")

	reviewerUser := createTestUser(t, "REVIEWER")
	reviewerToken := loginAs(t, e, reviewerUser.Username, "TestPass1!")

	// Create complete candidate.
	createBody, _ := json.Marshal(map[string]interface{}{
		"demographics":       map[string]string{"name": "John Doe"},
		"examScores":         map[string]int{"math": 95},
		"applicationDetails": map[string]string{"program": "CS"},
	})
	createRec := makeRequest(t, e, http.MethodPost, "/api/candidates", createBody, intakeToken)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	candidateID := createResp.Data.ID

	// Upload a document before submitting.
	minimalPDF := []byte("%PDF-1.0\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF")
	docRec := makeDocumentUploadRequest(t, e, "/api/candidates/"+candidateID+"/documents", minimalPDF, "application/pdf", "test.pdf", intakeToken)
	require.Equal(t, http.StatusCreated, docRec.Code)

	// Submit.
	submitRec := makeRequest(t, e, http.MethodPost, "/api/candidates/"+candidateID+"/submit", nil, intakeToken)
	require.Equal(t, http.StatusOK, submitRec.Code)

	// Approve.
	approveBody, _ := json.Marshal(map[string]string{"comments": "looks good"})
	approveRec := makeRequest(t, e, http.MethodPost, "/api/candidates/"+candidateID+"/approve", approveBody, reviewerToken)
	assert.Equal(t, http.StatusOK, approveRec.Code)
}

func TestCandidateReject_RequiresComments(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	intakeUser := createTestUser(t, "INTAKE_SPECIALIST")
	intakeToken := loginAs(t, e, intakeUser.Username, "TestPass1!")

	reviewerUser := createTestUser(t, "REVIEWER")
	reviewerToken := loginAs(t, e, reviewerUser.Username, "TestPass1!")

	// Create and submit complete candidate.
	createBody, _ := json.Marshal(map[string]interface{}{
		"demographics":       map[string]string{"name": "Jane Doe"},
		"examScores":         map[string]int{"math": 70},
		"applicationDetails": map[string]string{"program": "Arts"},
	})
	createRec := makeRequest(t, e, http.MethodPost, "/api/candidates", createBody, intakeToken)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	candidateID := createResp.Data.ID

	// Upload a document before submitting.
	minimalPDF := []byte("%PDF-1.0\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF")
	makeDocumentUploadRequest(t, e, "/api/candidates/"+candidateID+"/documents", minimalPDF, "application/pdf", "test.pdf", intakeToken)

	makeRequest(t, e, http.MethodPost, "/api/candidates/"+candidateID+"/submit", nil, intakeToken)

	// Reject with empty comments – should fail.
	rejectBody, _ := json.Marshal(map[string]string{"comments": ""})
	rec := makeRequest(t, e, http.MethodPost, "/api/candidates/"+candidateID+"/reject", rejectBody, reviewerToken)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCandidateWithoutAuth(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	rec := makeRequest(t, e, http.MethodGet, "/api/candidates", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
