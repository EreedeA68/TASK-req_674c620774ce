package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	internalDB "github.com/keystone/backend/internal/db"
	"github.com/keystone/backend/internal/audit"
	"github.com/keystone/backend/internal/auth"
	"github.com/keystone/backend/internal/candidate"
	"github.com/keystone/backend/internal/documents"
	"github.com/keystone/backend/internal/lostfound"
	appMiddleware "github.com/keystone/backend/internal/middleware"
	"github.com/keystone/backend/internal/parts"
	"github.com/keystone/backend/internal/reports"
	"github.com/keystone/backend/internal/search"
)

// APIResponse represents the standard response envelope.
type APIResponse struct {
	Data         interface{} `json:"data"`
	ErrorMessage string      `json:"errorMessage"`
	Code         int         `json:"code"`
	Details      interface{} `json:"details"`
}

var (
	testEcho *echo.Echo
	testDB   *gorm.DB
)

// setupTestServer initialises Echo with an in-memory test database.
// Returns the Echo instance; also sets the package-level testEcho variable.
func setupTestServer(t *testing.T) *echo.Echo {
	t.Helper()

	// Use a test DSN (or SQLite via an alternate driver if postgres isn't available).
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "postgres"),
			getEnv("TEST_DB_NAME", "keystone_test"),
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("skipping integration tests: could not connect to test database: %v", err)
		return nil
	}

	// Auto-migrate test schema.
	_ = db.AutoMigrate(
		&internalDB.User{},
		&internalDB.Session{},
		&internalDB.Candidate{},
		&internalDB.CandidateDocument{},
		&internalDB.Listing{},
		&internalDB.Part{},
		&internalDB.PartVersion{},
		&internalDB.AuditLog{},
		&internalDB.DownloadPermission{},
		&internalDB.DownloadLog{},
	)

	testDB = db
	internalDB.DB = db

	// Build dependencies.
	auditRepo := audit.NewRepository(db)
	auditSvc := audit.NewService(auditRepo)
	auditHandler := audit.NewHandler(auditSvc)

	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, auditSvc)
	authHandler := auth.NewHandler(authSvc)

	candidateRepo := candidate.NewRepository(db)
	candidateSvc := candidate.NewService(candidateRepo, auditSvc)
	candidateHandler := candidate.NewHandler(candidateSvc)

	listingRepo := lostfound.NewRepository(db)
	listingSvc := lostfound.NewService(listingRepo, auditSvc)
	listingHandler := lostfound.NewHandler(listingSvc)

	partsRepo := parts.NewRepository(db)
	partsSvc := parts.NewService(partsRepo, auditSvc)
	partsHandler := parts.NewHandler(partsSvc)

	docsSvc := documents.NewService(db, auditSvc)
	docsHandler := documents.NewHandler(docsSvc)

	searchSvc := search.NewService(db)
	searchHandler := search.NewHandler(searchSvc)

	reportsSvc := reports.NewService(db, auditSvc)
	reportsHandler := reports.NewHandler(reportsSvc)

	e := echo.New()
	e.HideBanner = true
	e.Use(echoMiddleware.Recover())

	registerTestRoutes(e, db, authHandler, candidateHandler, listingHandler, partsHandler, docsHandler, searchHandler, reportsHandler, auditHandler)

	testEcho = e
	return e
}

func registerTestRoutes(
	e *echo.Echo,
	db *gorm.DB,
	authHandler *auth.Handler,
	candidateHandler *candidate.Handler,
	listingHandler *lostfound.Handler,
	partsHandler *parts.Handler,
	docsHandler *documents.Handler,
	searchHandler *search.Handler,
	reportsHandler *reports.Handler,
	auditHandler *audit.Handler,
) {
	jwtMiddleware := appMiddleware.ValidateJWTWithDB(db)

	api := e.Group("/api")

	api.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data": map[string]string{"status": "ok"}, "errorMessage": "", "code": 200, "details": nil,
		})
	})

	authGroup := api.Group("/auth")
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/logout", authHandler.Logout, jwtMiddleware)
	authGroup.GET("/me", authHandler.Me, jwtMiddleware)
	authGroup.POST("/mfa/setup", authHandler.SetupMFA, jwtMiddleware)
	authGroup.POST("/mfa/verify", authHandler.VerifyMFA, jwtMiddleware)

	users := api.Group("/users", jwtMiddleware, appMiddleware.RequireRole("ADMIN"))
	users.POST("", authHandler.CreateUser)
	users.GET("", authHandler.ListUsers)

	candidates := api.Group("/candidates", jwtMiddleware)
	candidates.POST("", candidateHandler.Create, appMiddleware.RequireRole("ADMIN", "INTAKE_SPECIALIST"))
	candidates.GET("", candidateHandler.List, appMiddleware.RequireRole("ADMIN", "INTAKE_SPECIALIST", "REVIEWER", "AUDITOR"))
	candidates.GET("/:id", candidateHandler.Get, appMiddleware.RequireRole("ADMIN", "INTAKE_SPECIALIST", "REVIEWER", "AUDITOR"))
	candidates.PUT("/:id", candidateHandler.Update, appMiddleware.RequireRole("ADMIN", "INTAKE_SPECIALIST"))
	candidates.POST("/:id/submit", candidateHandler.Submit, appMiddleware.RequireRole("ADMIN", "INTAKE_SPECIALIST"))
	candidates.POST("/:id/approve", candidateHandler.Approve, appMiddleware.RequireRole("ADMIN", "REVIEWER"))
	candidates.POST("/:id/reject", candidateHandler.Reject, appMiddleware.RequireRole("ADMIN", "REVIEWER"))
	candidates.POST("/:id/documents", candidateHandler.UploadDocument, appMiddleware.RequireRole("ADMIN", "INTAKE_SPECIALIST"))
	candidates.GET("/:id/documents", candidateHandler.ListDocuments, appMiddleware.RequireRole("ADMIN", "INTAKE_SPECIALIST", "REVIEWER", "AUDITOR"))

	listings := api.Group("/listings", jwtMiddleware)
	listings.POST("", listingHandler.Create, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	listings.GET("", listingHandler.List)
	listings.GET("/:id", listingHandler.Get)
	listings.PUT("/:id", listingHandler.Update, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	listings.POST("/:id/unlist", listingHandler.Unlist, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "REVIEWER"))
	listings.DELETE("/:id", listingHandler.Delete, appMiddleware.RequireRole("ADMIN"))
	listings.POST("/:id/override-duplicate", listingHandler.OverrideDuplicate, appMiddleware.RequireRole("ADMIN", "REVIEWER"))

	partsGroup := api.Group("/parts", jwtMiddleware)
	partsGroup.POST("", partsHandler.Create, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	partsGroup.GET("", partsHandler.List, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "AUDITOR"))
	partsGroup.GET("/export", partsHandler.Export, appMiddleware.RequireRole("ADMIN", "AUDITOR", "INVENTORY_CLERK"))
	partsGroup.POST("/import", partsHandler.Import, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	partsGroup.GET("/:id", partsHandler.Get, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "AUDITOR"))
	partsGroup.PUT("/:id", partsHandler.Update, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	partsGroup.GET("/:id/versions", partsHandler.GetVersions, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "AUDITOR"))
	partsGroup.GET("/:id/versions/:v1/compare/:v2", partsHandler.CompareVersions, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "AUDITOR"))
	partsGroup.POST("/:id/promote", partsHandler.Promote, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))

	api.GET("/documents/:id/download", docsHandler.Download, jwtMiddleware, appMiddleware.RequirePermission("documents", "download"))

	api.GET("/search", searchHandler.Search, jwtMiddleware)

	reportsGroup := api.Group("/reports", jwtMiddleware, appMiddleware.RequireRole("ADMIN", "AUDITOR"))
	reportsGroup.GET("/kpi", reportsHandler.GetKPI, appMiddleware.RequirePermission("reports", "view"))
	reportsGroup.GET("/export", reportsHandler.Export, appMiddleware.RequirePermission("reports", "export"))

	api.GET("/audit-logs", auditHandler.GetLogs, jwtMiddleware, appMiddleware.RequireRole("ADMIN", "AUDITOR"))
}

// createTestUser inserts a user with the given role directly into the test DB.
func createTestUser(t *testing.T, role string) *internalDB.User {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte("TestPass1!"), bcrypt.MinCost)
	user := &internalDB.User{
		Username:     fmt.Sprintf("%s_user_%d", role, uniqueSuffix()),
		Email:        fmt.Sprintf("%s_%d@test.com", role, uniqueSuffix()),
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := testDB.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	t.Cleanup(func() {
		testDB.Delete(user)
	})
	return user
}

// loginAs performs a POST /api/auth/login and returns the JWT token.
func loginAs(t *testing.T, e *echo.Echo, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	rec := makeRequest(t, e, http.MethodPost, "/api/auth/login", body, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("login failed with status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.Data.Token
}

// makeRequest creates an httptest.ResponseRecorder for a request to the Echo server.
func makeRequest(t *testing.T, e *echo.Echo, method, path string, body []byte, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// makeDocumentUploadRequest creates a multipart/form-data request for document upload.
func makeDocumentUploadRequest(t *testing.T, e *echo.Echo, path string, fileContent []byte, mimeType, fileName, token string) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+fileName+`"`)
	h.Set("Content-Type", mimeType)
	fw, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create form part: %v", err)
	}
	if _, err := fw.Write(fileContent); err != nil {
		t.Fatalf("failed to write file content: %v", err)
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, path, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// parseResponse parses a standard API response.
func parseResponse(t *testing.T, rec *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var resp APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nbody: %s", err, rec.Body.String())
	}
	return resp
}

// getEnv returns the env var or a default value.
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

var counter int64

func uniqueSuffix() int64 {
	counter++
	return counter
}

// Note: counter is not atomic; tests should run serially (no t.Parallel()) to avoid races.
