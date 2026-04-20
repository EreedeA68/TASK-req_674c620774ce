package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/robfig/cron/v3"

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

func main() {
	// Fail fast if required secrets are absent.
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable must be set")
	}
	aesKey := os.Getenv("AES_KEY")
	if len(aesKey) != 32 {
		log.Fatalf("AES_KEY must be exactly 32 bytes, got %d", len(aesKey))
	}

	// Connect to PostgreSQL.
	db, err := internalDB.Connect()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Println("connected to database")

	// Build the dependency tree.
	auditRepo := audit.NewRepository(db)
	auditSvc := audit.NewService(auditRepo)

	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, auditSvc)
	authHandler := auth.NewHandler(authSvc)
	adminHandler := auth.NewAdminHandler(authSvc)

	candidateRepo := candidate.NewRepository(db)
	candidateSvc := candidate.NewService(candidateRepo, auditSvc)
	candidateHandler := candidate.NewHandler(candidateSvc)

	listingRepo := lostfound.NewRepository(db)
	listingSvc := lostfound.NewService(listingRepo, auditSvc)
	listingHandler := lostfound.NewHandler(listingSvc)

	partsRepo := parts.NewRepository(db)
	partsSvc := parts.NewService(partsRepo, auditSvc)
	partsHandler := parts.NewHandler(partsSvc)

	searchSvc := search.NewService(db)
	searchHandler := search.NewHandler(searchSvc)

	reportsSvc := reports.NewService(db, auditSvc)
	reportsHandler := reports.NewHandler(reportsSvc)

	docsSvc := documents.NewService(db, auditSvc)
	docsHandler := documents.NewHandler(docsSvc)

	auditHandler := audit.NewHandler(auditSvc)

	// Setup Echo.
	e := echo.New()
	e.HideBanner = true

	// Global middleware.
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var corsOrigins []string
	if allowedOrigins != "" {
		corsOrigins = strings.Split(allowedOrigins, ",")
	} else {
		corsOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
	}
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: corsOrigins,
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodDelete, http.MethodOptions, http.MethodPatch,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin, echo.HeaderContentType,
			echo.HeaderAccept, echo.HeaderAuthorization,
			"X-Device-ID",
		},
	}))

	// Build a session-aware JWT middleware (checks that the session is not invalidated in DB).
	jwtMiddleware := appMiddleware.ValidateJWTWithDB(db)

	// Register all routes.
	registerRoutes(e, db, jwtMiddleware, authHandler, adminHandler, candidateHandler, listingHandler, partsHandler, searchHandler, reportsHandler, auditHandler, docsHandler)

	// Start the daily auto-unlist cron job (runs at midnight).
	c := cron.New()
	_, err = c.AddFunc("0 0 * * *", func() {
		log.Println("[Cron] running auto-unlist job")
		listingSvc.RunAutoUnlistJob()
	})
	if err != nil {
		log.Printf("failed to register cron job: %v", err)
	}
	c.Start()
	defer c.Stop()

	// Start server.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("starting server on :%s", port)
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func registerRoutes(
	e *echo.Echo,
	_ interface{},
	jwtMiddleware echo.MiddlewareFunc,
	authHandler *auth.Handler,
	adminHandler *auth.AdminHandler,
	candidateHandler *candidate.Handler,
	listingHandler *lostfound.Handler,
	partsHandler *parts.Handler,
	searchHandler *search.Handler,
	reportsHandler *reports.Handler,
	auditHandler *audit.Handler,
	docsHandler *documents.Handler,
) {
	api := e.Group("/api")

	// Health check.
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":         map[string]string{"status": "ok"},
			"errorMessage": "",
			"code":         http.StatusOK,
			"details":      nil,
		})
	})

	// Auth (no JWT required for login/logout).
	authGroup := api.Group("/auth")
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/logout", authHandler.Logout, appMiddleware.ValidateJWT)
	authGroup.GET("/me", authHandler.Me, jwtMiddleware)
	authGroup.POST("/mfa/setup", authHandler.SetupMFA, jwtMiddleware)
	authGroup.POST("/mfa/verify", authHandler.VerifyMFA, jwtMiddleware)

	// Users (admin only).
	users := api.Group("/users", jwtMiddleware, appMiddleware.RequireRole("ADMIN"))
	users.POST("", authHandler.CreateUser)
	users.GET("", authHandler.ListUsers)

	// Admin panel endpoints (admin only).
	admin := api.Group("/admin", jwtMiddleware, appMiddleware.RequireRole("ADMIN"))
	admin.GET("/users", adminHandler.ListUsers)
	admin.POST("/users", adminHandler.CreateUser)

	// Candidates.
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

	// Listings – mutating endpoints are role-restricted.
	listings := api.Group("/listings", jwtMiddleware)
	listings.POST("", listingHandler.Create, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	listings.GET("", listingHandler.List)
	listings.GET("/:id", listingHandler.Get)
	listings.PUT("/:id", listingHandler.Update, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	listings.POST("/:id/unlist", listingHandler.Unlist, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "REVIEWER"))
	listings.DELETE("/:id", listingHandler.Delete, appMiddleware.RequireRole("ADMIN"))
	listings.POST("/:id/override-duplicate", listingHandler.OverrideDuplicate, appMiddleware.RequireRole("ADMIN", "REVIEWER"))

	// Parts.
	partsGroup := api.Group("/parts", jwtMiddleware)
	partsGroup.POST("", partsHandler.Create, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	partsGroup.GET("", partsHandler.List, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "AUDITOR"))
	partsGroup.GET("/export", partsHandler.Export, appMiddleware.RequireRole("ADMIN", "AUDITOR", "INVENTORY_CLERK"), appMiddleware.RequirePermission("parts", "export"))
	partsGroup.POST("/import", partsHandler.Import, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	partsGroup.GET("/:id", partsHandler.Get, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "AUDITOR"))
	partsGroup.PUT("/:id", partsHandler.Update, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))
	partsGroup.GET("/:id/versions", partsHandler.GetVersions, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "AUDITOR"))
	partsGroup.GET("/:id/versions/:v1/compare/:v2", partsHandler.CompareVersions, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK", "AUDITOR"))
	partsGroup.POST("/:id/promote", partsHandler.Promote, appMiddleware.RequireRole("ADMIN", "INVENTORY_CLERK"))

	// Search.
	api.GET("/search", searchHandler.Search, jwtMiddleware)

	// Reports – restricted to ADMIN and AUDITOR.
	reportsGroup := api.Group("/reports", jwtMiddleware, appMiddleware.RequireRole("ADMIN", "AUDITOR"))
	reportsGroup.GET("/kpi", reportsHandler.GetKPI, appMiddleware.RequirePermission("reports", "view"))
	reportsGroup.GET("/export", reportsHandler.Export, appMiddleware.RequirePermission("reports", "export"))

	// Audit logs (AUDITOR and ADMIN only).
	api.GET("/audit-logs", auditHandler.GetLogs, jwtMiddleware, appMiddleware.RequireRole("ADMIN", "AUDITOR"))

	// Documents.
	api.GET("/documents/:id/download", docsHandler.Download, jwtMiddleware, appMiddleware.RequirePermission("documents", "download"))
}
