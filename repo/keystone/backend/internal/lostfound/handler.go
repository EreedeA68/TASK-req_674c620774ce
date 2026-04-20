package lostfound

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	appMiddleware "github.com/keystone/backend/internal/middleware"
	"gorm.io/gorm"
)

// Handler handles HTTP requests for lost & found endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new lostfound Handler.
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

// Create handles POST /api/listings.
func (h *Handler) Create(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	var req CreateListingRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	dto, err := h.svc.CreateListing(user.UserID, user.SiteID, user.OrganizationID, appMiddleware.GetDeviceID(c), c.RealIP(), req)
	if err != nil {
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusCreated, dto, "", nil)
}

// List handles GET /api/listings.
func (h *Handler) List(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	role := ""
	if user != nil {
		role = user.Role
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
		"category": c.QueryParam("category"),
		"status":   c.QueryParam("status"),
	}
	if user != nil && user.OrganizationID != "" {
		filters["orgID"] = user.OrganizationID
	}
	if user != nil && user.SiteID != "" {
		filters["siteID"] = user.SiteID
	}

	listings, total, err := h.svc.GetListings(role, filters, page, limit)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusOK, ListingListResponse{
		Items: listings,
		Total: total,
		Page:  page,
		Limit: limit,
	}, "", nil)
}

// Get handles GET /api/listings/:id.
func (h *Handler) Get(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	role := ""
	if user != nil {
		role = user.Role
	}
	id := c.Param("id")
	orgID := ""
	siteID := ""
	if user != nil {
		orgID = user.OrganizationID
		siteID = user.SiteID
	}
	dto, err := h.svc.GetListing(id, role, orgID, siteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || err.Error() == "listing not found" {
			return respond(c, http.StatusNotFound, nil, "listing not found", nil)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, dto, "", nil)
}

// Update handles PUT /api/listings/:id.
func (h *Handler) Update(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	var req CreateListingRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	dto, err := h.svc.EditListing(id, user.UserID, user.Role, appMiddleware.GetDeviceID(c), c.RealIP(), req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "listing not found", nil)
		}
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, dto, "", nil)
}

// Unlist handles POST /api/listings/:id/unlist.
func (h *Handler) Unlist(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	if err := h.svc.UnlistListing(id, user.UserID, appMiddleware.GetDeviceID(c), c.RealIP()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "listing not found", nil)
		}
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, map[string]string{"message": "listing unlisted"}, "", nil)
}

// Delete handles DELETE /api/listings/:id.
func (h *Handler) Delete(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	if err := h.svc.DeleteListing(id, user.UserID, appMiddleware.GetDeviceID(c), c.RealIP()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "listing not found", nil)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, map[string]string{"message": "listing deleted"}, "", nil)
}

// OverrideDuplicate handles POST /api/listings/:id/override-duplicate.
func (h *Handler) OverrideDuplicate(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	id := c.Param("id")
	if err := h.svc.OverrideDuplicate(id, user.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return respond(c, http.StatusNotFound, nil, "listing not found", nil)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, map[string]string{"message": "duplicate flag cleared"}, "", nil)
}
