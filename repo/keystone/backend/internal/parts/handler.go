package parts

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	appMiddleware "github.com/keystone/backend/internal/middleware"
	"gorm.io/gorm"
)

// Handler handles HTTP requests for parts endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new parts Handler.
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

// Create handles POST /api/parts.
func (h *Handler) Create(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	var req CreatePartRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	dto, err := h.svc.CreatePart(user.UserID, user.SiteID, user.OrganizationID, appMiddleware.GetDeviceID(c), c.RealIP(), req)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "failed to create part", nil)
	}
	return respond(c, http.StatusCreated, dto, "", nil)
}

// List handles GET /api/parts.
func (h *Handler) List(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filters := map[string]string{
		"status": c.QueryParam("status"),
		"search": c.QueryParam("search"),
	}
	if user != nil && user.OrganizationID != "" {
		filters["orgID"] = user.OrganizationID
	}
	if user != nil && user.SiteID != "" {
		filters["siteID"] = user.SiteID
	}

	parts, total, err := h.svc.GetParts(filters, page, limit)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusOK, PartListResponse{
		Items: parts,
		Total: total,
		Page:  page,
		Limit: limit,
	}, "", nil)
}

// Get handles GET /api/parts/:id.
func (h *Handler) Get(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}
	id := c.Param("id")
	dto, err := h.svc.GetPart(id, user.OrganizationID, user.SiteID, user.Role)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "part not found", nil)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, dto, "", nil)
}

// Update handles PUT /api/parts/:id.
func (h *Handler) Update(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	var req CreatePartRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	dto, err := h.svc.UpdatePart(id, user.UserID, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "part not found", nil)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, dto, "", nil)
}

// GetVersions handles GET /api/parts/:id/versions.
func (h *Handler) GetVersions(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}
	id := c.Param("id")
	dtos, err := h.svc.GetVersions(id, user.OrganizationID, user.SiteID, user.Role)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "part not found", nil)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, dtos, "", nil)
}

// CompareVersions handles GET /api/parts/:id/versions/:v1/compare/:v2.
func (h *Handler) CompareVersions(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}
	id := c.Param("id")
	v1, _ := strconv.Atoi(c.Param("v1"))
	v2, _ := strconv.Atoi(c.Param("v2"))

	result, err := h.svc.CompareVersions(id, user.OrganizationID, user.SiteID, user.Role, v1, v2)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "part not found", nil)
		}
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, result, "", nil)
}

// Promote handles POST /api/parts/:id/promote.
func (h *Handler) Promote(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	partID := c.Param("id")
	var body struct {
		VersionID string `json:"versionId"`
	}
	if err := c.Bind(&body); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	if err := h.svc.PromoteVersion(partID, body.VersionID, user.UserID); err != nil {
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, map[string]string{"message": "version promoted"}, "", nil)
}

// Import handles POST /api/parts/import (multipart CSV upload).
func (h *Handler) Import(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return respond(c, http.StatusBadRequest, nil, "CSV file is required", nil)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "failed to open file", nil)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "failed to read file", nil)
	}

	rows, err := ParseCSVRows(data)
	if err != nil {
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}

	// If preview=true, validate rows and return results without persisting.
	if c.QueryParam("preview") == "true" {
		validationErrors := h.svc.ValidateRows(rows)
		return respond(c, http.StatusOK, map[string]interface{}{
			"rows":             rows,
			"count":            len(rows),
			"validationErrors": validationErrors,
		}, "", nil)
	}

	count, validationErrors, err := h.svc.BulkImport(rows, user.UserID)
	if err != nil {
		if len(validationErrors) > 0 {
			return respond(c, http.StatusUnprocessableEntity, nil, "validation failed", validationErrors)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusOK, map[string]interface{}{
		"imported": count,
		"message":  "import successful",
	}, "", nil)
}

// Export handles GET /api/parts/export.
func (h *Handler) Export(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	fieldsParam := c.QueryParam("fields")
	var fields []string
	if fieldsParam != "" {
		fields = strings.Split(fieldsParam, ",")
	}

	csvData, err := h.svc.ExportCSV(user.UserID, appMiddleware.GetDeviceID(c), c.RealIP(), fields)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "export failed", nil)
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=parts_export.csv")
	c.Response().Header().Set("Content-Type", "text/csv")
	return c.Blob(http.StatusOK, "text/csv", csvData)
}
