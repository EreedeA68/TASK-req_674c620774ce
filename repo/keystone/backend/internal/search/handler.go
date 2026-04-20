package search

import (
	"net/http"

	"github.com/labstack/echo/v4"
	appMiddleware "github.com/keystone/backend/internal/middleware"
)

// Handler handles HTTP requests for search endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new search Handler.
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

// Search handles GET /api/search.
func (h *Handler) Search(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	role := ""
	callerID := ""
	orgID := ""
	siteID := ""
	if user != nil {
		role = user.Role
		callerID = user.UserID
		orgID = user.OrganizationID
		siteID = user.SiteID
	}

	query := c.QueryParam("q")
	fuzzy := c.QueryParam("fuzzy") == "true"

	results, err := h.svc.Search(query, fuzzy, role, callerID, orgID, siteID)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "search failed", nil)
	}

	return respond(c, http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
	}, "", nil)
}
