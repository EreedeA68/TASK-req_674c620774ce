package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validPartBody() []byte {
	body, _ := json.Marshal(map[string]interface{}{
		"partNumber":    fmt.Sprintf("PN-%d", uniqueSuffix()),
		"name":          "Brake Pad",
		"description":   "Heavy duty brake pad",
		"fitment":       map[string]interface{}{"make": "Ford", "model": "F-150", "yearStart": 2010, "yearEnd": 2020},
		"oemMappings":   map[string]string{"ford": "F123"},
		"attributes":    map[string]string{"material": "ceramic"},
		"changeSummary": "Initial version",
	})
	return body
}

func TestPartCreate_AsInventoryClerk(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodPost, "/api/parts", validPartBody(), token)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp struct {
		Data struct {
			ID           string `json:"id"`
			PartNumber   string `json:"partNumber"`
			Name         string `json:"name"`
			Status       string `json:"status"`
			VersionNumber int   `json:"versionNumber"`
		} `json:"data"`
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.ID)
	assert.Equal(t, "ACTIVE", resp.Data.Status)
	assert.Equal(t, 1, resp.Data.VersionNumber)
	assert.Equal(t, 201, resp.Code)
}

func TestPartCreate_AsReviewer_Forbidden(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodPost, "/api/parts", validPartBody(), token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestPartList(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/parts", nil, token)
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

func TestPartGet_NotFound(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/parts/00000000-0000-0000-0000-000000000000", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPartUpdate_CreatesNewVersion(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Create part.
	createRec := makeRequest(t, e, http.MethodPost, "/api/parts", validPartBody(), token)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct {
			ID         string `json:"id"`
			PartNumber string `json:"partNumber"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))

	// Update part.
	updateBody, _ := json.Marshal(map[string]interface{}{
		"partNumber":    createResp.Data.PartNumber,
		"name":          "Brake Pad v2",
		"description":   "Updated description",
		"fitment":       map[string]interface{}{"make": "Toyota"},
		"oemMappings":   map[string]string{"toyota": "T456"},
		"attributes":    map[string]string{"material": "metallic"},
		"changeSummary": "Updated fitment",
	})
	updateRec := makeRequest(t, e, http.MethodPut, "/api/parts/"+createResp.Data.ID, updateBody, token)
	require.Equal(t, http.StatusOK, updateRec.Code)

	var updateResp struct {
		Data struct {
			VersionNumber int `json:"versionNumber"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(updateRec.Body.Bytes(), &updateResp))
	assert.Equal(t, 2, updateResp.Data.VersionNumber)
}

func TestPartGetVersions(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Create part.
	createRec := makeRequest(t, e, http.MethodPost, "/api/parts", validPartBody(), token)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))

	rec := makeRequest(t, e, http.MethodGet, "/api/parts/"+createResp.Data.ID+"/versions", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			VersionNumber int `json:"versionNumber"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, 1, resp.Data[0].VersionNumber)
}

func TestPartImport_ValidCSV(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	csvContent := fmt.Sprintf("part_number,name,description\nPN-CSV-%d,CSV Part,CSV Description\n", uniqueSuffix())

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, _ := w.CreateFormFile("file", "parts.csv")
	fw.Write([]byte(csvContent))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/parts/import", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Imported int `json:"imported"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Data.Imported)
}

func TestPartExport(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	rec := makeRequest(t, e, http.MethodGet, "/api/parts/export", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/csv", rec.Header().Get("Content-Type"))
}

func TestPartCompareVersions(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INVENTORY_CLERK")
	token := loginAs(t, e, user.Username, "TestPass1!")

	// Create part (version 1).
	createRec := makeRequest(t, e, http.MethodPost, "/api/parts", validPartBody(), token)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp struct {
		Data struct {
			ID         string `json:"id"`
			PartNumber string `json:"partNumber"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))

	// Update (version 2).
	updateBody, _ := json.Marshal(map[string]interface{}{
		"partNumber":    createResp.Data.PartNumber,
		"name":          "Updated Brake Pad",
		"description":   "Updated",
		"fitment":       map[string]interface{}{"make": "GM"},
		"oemMappings":   map[string]string{"gm": "G789"},
		"attributes":    map[string]string{"material": "composite"},
		"changeSummary": "Changed fitment",
	})
	makeRequest(t, e, http.MethodPut, "/api/parts/"+createResp.Data.ID, updateBody, token)

	// Compare v1 with v2.
	rec := makeRequest(t, e, http.MethodGet, "/api/parts/"+createResp.Data.ID+"/versions/1/compare/2", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			VersionA struct{ VersionNumber int `json:"versionNumber"` } `json:"versionA"`
			VersionB struct{ VersionNumber int `json:"versionNumber"` } `json:"versionB"`
			Diff     map[string]interface{} `json:"diff"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Data.VersionA.VersionNumber)
	assert.Equal(t, 2, resp.Data.VersionB.VersionNumber)
	assert.NotNil(t, resp.Data.Diff)
}
