package candidate

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	appMiddleware "github.com/keystone/backend/internal/middleware"
	"gorm.io/gorm"
)

// Handler handles HTTP requests for candidate endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new candidate Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func respond(c echo.Context, statusCode int, data interface{}, errMsg string, details interface{}) error {
	return c.JSON(statusCode, map[string]interface{}{
		"data":         data,
		"errorMessage": errMsg,
		"code":         statusCode,
		"details":      details,
	})
}

// Create handles POST /api/candidates.
func (h *Handler) Create(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	var req CreateCandidateRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	dto, err := h.svc.CreateDraft(user.UserID, user.SiteID, user.OrganizationID, appMiddleware.GetDeviceID(c), c.RealIP(), req)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "failed to create candidate", nil)
	}

	return respond(c, http.StatusCreated, dto, "", nil)
}

// List handles GET /api/candidates.
func (h *Handler) List(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filters := map[string]string{
		"status":    c.QueryParam("status"),
		"createdBy": c.QueryParam("createdBy"),
	}
	if user.OrganizationID != "" {
		filters["orgID"] = user.OrganizationID
	}
	if user.SiteID != "" {
		filters["siteID"] = user.SiteID
	}

	candidates, total, err := h.svc.GetCandidates(user.UserID, user.Role, filters, page, limit)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusOK, CandidateListResponse{
		Items: candidates,
		Total: total,
		Page:  page,
		Limit: limit,
	}, "", nil)
}

// Get handles GET /api/candidates/:id.
func (h *Handler) Get(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	dto, err := h.svc.GetCandidate(id, user.UserID, user.Role, user.OrganizationID, user.SiteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "candidate not found", nil)
		}
		if err.Error() == "not authorized to view this candidate" {
			return respond(c, http.StatusForbidden, nil, err.Error(), nil)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, dto, "", nil)
}

// Update handles PUT /api/candidates/:id.
func (h *Handler) Update(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")

	var req CreateCandidateRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	dto, err := h.svc.UpdateDraft(id, user.UserID, user.Role, appMiddleware.GetDeviceID(c), c.RealIP(), req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "candidate not found", nil)
		}
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, dto, "", nil)
}

// Submit handles POST /api/candidates/:id/submit.
func (h *Handler) Submit(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	if err := h.svc.Submit(id, user.UserID, user.Role, appMiddleware.GetDeviceID(c), c.RealIP()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "candidate not found", nil)
		}
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, map[string]string{"message": "candidate submitted"}, "", nil)
}

// Approve handles POST /api/candidates/:id/approve.
func (h *Handler) Approve(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	var req ApproveRejectRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	if err := h.svc.Approve(id, user.UserID, req.Comments, appMiddleware.GetDeviceID(c), c.RealIP()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "candidate not found", nil)
		}
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, map[string]string{"message": "candidate approved"}, "", nil)
}

// Reject handles POST /api/candidates/:id/reject.
func (h *Handler) Reject(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	var req ApproveRejectRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	if err := h.svc.Reject(id, user.UserID, req.Comments, appMiddleware.GetDeviceID(c), c.RealIP()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "candidate not found", nil)
		}
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, map[string]string{"message": "candidate rejected"}, "", nil)
}

// UploadDocument handles POST /api/candidates/:id/documents.
func (h *Handler) UploadDocument(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	candidateID := c.Param("id")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return respond(c, http.StatusBadRequest, nil, "file is required", nil)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "failed to open file", nil)
	}
	defer file.Close()

	dto, err := h.svc.UploadDocument(candidateID, user.UserID, user.Role, appMiddleware.GetDeviceID(c), c.RealIP(), file, fileHeader)
	if err != nil {
		switch err.Error() {
		case "not authorized to upload documents for this candidate":
			return respond(c, http.StatusForbidden, nil, err.Error(), nil)
		case "duplicate document: file already exists":
			return respond(c, http.StatusConflict, nil, err.Error(), nil)
		case "unsupported file type; only PDF, JPG, PNG allowed":
			return respond(c, http.StatusUnprocessableEntity, nil, err.Error(), nil)
		default:
			return respond(c, http.StatusInternalServerError, nil, "failed to process upload", nil)
		}
	}
	return respond(c, http.StatusCreated, dto, "", nil)
}

// ListDocuments handles GET /api/candidates/:id/documents.
func (h *Handler) ListDocuments(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	candidateID := c.Param("id")
	// Verify the caller can access this candidate before listing its documents.
	if _, err := h.svc.GetCandidate(candidateID, user.UserID, user.Role, user.OrganizationID, user.SiteID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "candidate not found", nil)
		}
		return respond(c, http.StatusForbidden, nil, "access denied", nil)
	}

	docs, err := h.svc.GetDocuments(candidateID)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "failed to retrieve documents", nil)
	}
	return respond(c, http.StatusOK, docs, "", nil)
}
