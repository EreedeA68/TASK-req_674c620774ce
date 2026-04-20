package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthLogin_ValidCredentials(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")
	assert.NotEmpty(t, token, "login should return a JWT token")
}

func TestAuthLogin_InvalidPassword(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	body, _ := json.Marshal(map[string]string{
		"username": user.Username,
		"password": "WrongPassword!",
	})
	rec := makeRequest(t, e, http.MethodPost, "/api/auth/login", body, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	resp := parseResponse(t, rec)
	assert.NotEmpty(t, resp.ErrorMessage)
}

func TestAuthLogin_UnknownUser(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	body, _ := json.Marshal(map[string]string{
		"username": "nonexistent_user_xyz",
		"password": "SomePassword1!",
	})
	rec := makeRequest(t, e, http.MethodPost, "/api/auth/login", body, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthLogin_ResponseStructure(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "ADMIN")
	body, _ := json.Marshal(map[string]string{
		"username": user.Username,
		"password": "TestPass1!",
	})
	rec := makeRequest(t, e, http.MethodPost, "/api/auth/login", body, "")
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Token           string   `json:"token"`
			Role            string   `json:"role"`
			MenuPermissions []string `json:"menuPermissions"`
			User            struct {
				ID         string `json:"id"`
				Username   string `json:"username"`
				Email      string `json:"email"`
				Role       string `json:"role"`
				MFAEnabled bool   `json:"mfaEnabled"`
			} `json:"user"`
		} `json:"data"`
		ErrorMessage string `json:"errorMessage"`
		Code         int    `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp.Data.Token)
	assert.Equal(t, "ADMIN", resp.Data.Role)
	assert.NotEmpty(t, resp.Data.MenuPermissions)
	assert.Equal(t, user.Username, resp.Data.User.Username)
	assert.Equal(t, 200, resp.Code)
	assert.Empty(t, resp.ErrorMessage)
}

func TestAuthMe_WithToken(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "REVIEWER")
	token := loginAs(t, e, user.Username, "TestPass1!")
	require.NotEmpty(t, token)

	rec := makeRequest(t, e, http.MethodGet, "/api/auth/me", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, user.Username, resp.Data.Username)
	assert.Equal(t, "REVIEWER", resp.Data.Role)
}

func TestAuthMe_WithoutToken(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	rec := makeRequest(t, e, http.MethodGet, "/api/auth/me", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthLogout(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")
	require.NotEmpty(t, token)

	rec := makeRequest(t, e, http.MethodPost, "/api/auth/logout", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMFASetup(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")
	token := loginAs(t, e, user.Username, "TestPass1!")
	require.NotEmpty(t, token)

	rec := makeRequest(t, e, http.MethodPost, "/api/auth/mfa/setup", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Secret string `json:"secret"`
			QRData string `json:"qrData"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.Secret)
	assert.Contains(t, resp.Data.QRData, "otpauth://")
}

func TestAuthAccountLockAfterFiveFailures(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	user := createTestUser(t, "INTAKE_SPECIALIST")

	// Make 5 wrong-password attempts.
	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(map[string]string{
			"username": user.Username,
			"password": "WrongPassword!",
		})
		makeRequest(t, e, http.MethodPost, "/api/auth/login", body, "")
	}

	// 6th attempt should be locked.
	body, _ := json.Marshal(map[string]string{
		"username": user.Username,
		"password": "WrongPassword!",
	})
	rec := makeRequest(t, e, http.MethodPost, "/api/auth/login", body, "")
	assert.Equal(t, http.StatusLocked, rec.Code)
}

func TestHealthCheck(t *testing.T) {
	e := setupTestServer(t)
	if e == nil {
		return
	}

	rec := makeRequest(t, e, http.MethodGet, "/api/health", nil, "")
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp.Data.Status)
}
