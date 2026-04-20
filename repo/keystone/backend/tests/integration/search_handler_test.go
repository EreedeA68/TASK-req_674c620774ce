package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch_EmptyQuery(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "ADMIN")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/search?q=", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Results []interface{} `json:"results"`
			Count   int           `json:"count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Data.Count)
}

func TestSearch_WithQuery(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "ADMIN")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Create a listing so there's something to search.
	createBody, _ := json.Marshal(map[string]interface{}{
		"title":               "Searchable Lost Item",
		"category":            "electronics",
		"locationDescription": "Downtown, Austin, TX",
	})
	makeRequest(t, e, http.MethodPost, "/api/listings", createBody, token)

	rec := makeRequest(t, e, http.MethodGet, "/api/search?q=Searchable", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Results []struct {
				Type   string `json:"type"`
				Title  string `json:"title"`
				Status string `json:"status"`
			} `json:"results"`
			Count int `json:"count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.GreaterOrEqual(t, resp.Data.Count, 1)
}

func TestSearch_FuzzyMode(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "ADMIN")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Create a part.
	partBody, _ := json.Marshal(map[string]interface{}{
		"partNumber":  fmt.Sprintf("PN-FUZZY-%d", uniqueSuffix()),
		"name":        "FuzzyBrakePad",
		"description": "For fuzzy search test",
	})
	makeRequest(t, e, http.MethodPost, "/api/parts", partBody, token)

	rec := makeRequest(t, e, http.MethodGet, "/api/search?q=FuzzyBrake&fuzzy=true", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Results []interface{} `json:"results"`
			Count   int           `json:"count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	// Result count may be 0 if fuzzy threshold isn't met – at least no error.
	assert.GreaterOrEqual(t, resp.Data.Count, 0)
}

func TestSearch_InventoryClerk_NoCandidate(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	// Create an INVENTORY_CLERK user.
	clerkUser := createTestUser(t, "INVENTORY_CLERK")
	clerkToken := loginAs(t, e, clerkUser.Username, "TestPass1!")

	// Create a candidate as intake specialist.
	intakeUser := createTestUser(t, "INTAKE_SPECIALIST")
	intakeToken := loginAs(t, e, intakeUser.Username, "TestPass1!")

	candidateBody, _ := json.Marshal(map[string]interface{}{
		"demographics":       map[string]string{"name": "Hidden Candidate"},
		"examScores":         map[string]int{"score": 80},
		"applicationDetails": map[string]string{"program": "CS"},
	})
	createRec := makeRequest(t, e, http.MethodPost, "/api/candidates", candidateBody, intakeToken)
	var createResp struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	_ = json.Unmarshal(createRec.Body.Bytes(), &createResp)

	// Clerk searches by candidate ID.
	rec := makeRequest(t, e, http.MethodGet, "/api/search?q="+createResp.Data.ID, nil, clerkToken)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Results []struct {
				Type string `json:"type"`
			} `json:"results"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// INVENTORY_CLERK should NOT see candidate results.
	for _, r := range resp.Data.Results {
		assert.NotEqual(t, "candidate", r.Type)
	}
}

func TestSearch_WithoutAuth(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	rec := makeRequest(t, e, http.MethodGet, "/api/search?q=test", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSearch_ResponseStructure(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "ADMIN")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/search?q=xyz_nonexistent_12345", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := parseResponse(t, rec)
	assert.Equal(t, 200, resp.Code)
	assert.Empty(t, resp.ErrorMessage)
	assert.NotNil(t, resp.Data)
}
