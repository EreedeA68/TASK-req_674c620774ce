package audit

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handler handles HTTP requests for audit endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new audit Handler.
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

// GetLogs handles GET /api/audit-logs.
func (h *Handler) GetLogs(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filters := map[string]string{
		"resourceType": c.QueryParam("resourceType"),
		"actorId":      c.QueryParam("actorId"),
		"action":       c.QueryParam("action"),
	}

	logs, total, err := h.svc.GetLogs(page, limit, filters)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusOK, map[string]interface{}{
		"items": logs,
		"total": total,
		"page":  page,
		"limit": limit,
	}, "", nil)
}
