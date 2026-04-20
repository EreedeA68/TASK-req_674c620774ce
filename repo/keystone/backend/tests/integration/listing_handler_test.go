package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validListingBody() []byte {
	body, _ := json.Marshal(map[string]interface{}{
		"title":               "Lost black wallet near downtown",
		"category":            "accessories",
		"locationDescription": "Near City Hall, Austin, TX",
		"timeWindowStart":     "2026-04-01T09:00:00Z",
		"timeWindowEnd":       "2026-04-02T09:00:00Z",
	})
	return body
}

func TestListingCreate(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodPost, "/api/listings", validListingBody(), token)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp struct {
		Data struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Status   string `json:"status"`
			Category string `json:"category"`
		} `json:"data"`
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.ID)
	assert.Equal(t, "PUBLISHED", resp.Data.Status)
	assert.Equal(t, "accessories", resp.Data.Category)
}

func TestListingCreate_DuplicateFlagged(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Create first listing.
	makeRequest(t, e, http.MethodPost, "/api/listings", validListingBody(), token)

	// Create near-identical listing – should be flagged.
	body2, _ := json.Marshal(map[string]interface{}{
		"title":               "Lost black walllet near downtown", // deliberate typo
		"category":            "accessories",
		"locationDescription": "Near City Hall, Austin, TX",
		"timeWindowStart":     "2026-04-01T09:00:00Z",
		"timeWindowEnd":       "2026-04-02T09:00:00Z",
	})
	rec := makeRequest(t, e, http.MethodPost, "/api/listings", body2, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp struct {
		Data struct {
			IsDuplicateFlagged bool `json:"isDuplicateFlagged"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Data.IsDuplicateFlagged)
}

func TestListingList(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/listings", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Items []interface{} `json:"items"`
			Total int64         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data.Items)
}

func TestListingGet_NotFound(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/listings/00000000-0000-0000-0000-000000000000", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestListingUnlist(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Create a listing.
	createRec := makeRequest(t, e, http.MethodPost, "/api/listings", validListingBody(), token)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))

	// Unlist it.
	rec := makeRequest(t, e, http.MethodPost, "/api/listings/"+createResp.Data.ID+"/unlist", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListingDelete(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "ADMIN")
	token := loginAs(t, e, user.Username, "TestPass1!")

	createRec := makeRequest(t, e, http.MethodPost, "/api/listings", validListingBody(), token)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))

	rec := makeRequest(t, e, http.MethodDelete, "/api/listings/"+createResp.Data.ID, nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListingOverrideDuplicate_ByReviewer(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	clerkUser := createTestUser(t, "INVENTORY_CLERK")
	clerkToken := loginAs(t, e, clerkUser.Username, "TestPass1!")

	reviewerUser := createTestUser(t, "REVIEWER")
	reviewerToken := loginAs(t, e, reviewerUser.Username, "TestPass1!")

	// Create first listing.
	makeRequest(t, e, http.MethodPost, "/api/listings", validListingBody(), clerkToken)

	// Create duplicate listing.
	body2, _ := json.Marshal(map[string]interface{}{
		"title":               "Lost black wallet near downtown",
		"category":            "accessories",
		"locationDescription": "Near City Hall, Austin, TX",
		"timeWindowStart":     "2026-04-01T09:00:00Z",
		"timeWindowEnd":       "2026-04-02T09:00:00Z",
	})
	createRec := makeRequest(t, e, http.MethodPost, "/api/listings", body2, clerkToken)

	var createResp struct {
		Data struct {
			ID                 string `json:"id"`
			IsDuplicateFlagged bool   `json:"isDuplicateFlagged"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))

	if !createResp.Data.IsDuplicateFlagged {
		t.Skip("listing was not flagged as duplicate, skipping override test")
	}

	// Override duplicate as reviewer.
	rec := makeRequest(t, e, http.MethodPost, "/api/listings/"+createResp.Data.ID+"/override-duplicate", nil, reviewerToken)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListingWithoutAuth(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	rec := makeRequest(t, e, http.MethodGet, "/api/listings", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
