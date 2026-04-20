package reports

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	appMiddleware "github.com/keystone/backend/internal/middleware"
)

// Handler handles HTTP requests for report endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new reports Handler.
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

// GetKPI handles GET /api/reports/kpi.
func (h *Handler) GetKPI(c echo.Context) error {
	kpi, err := h.svc.GetKPIs()
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "failed to retrieve KPIs", nil)
	}
	return respond(c, http.StatusOK, kpi, "", nil)
}

// Export handles GET /api/reports/export.
func (h *Handler) Export(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	role := ""
	actorID := ""
	if user != nil {
		role = user.Role
		actorID = user.UserID
	}

	fieldsParam := c.QueryParam("fields")
	var fields []string
	if fieldsParam != "" {
		fields = strings.Split(fieldsParam, ",")
	}

	csvData, err := h.svc.ExportReport(fields, role, actorID, appMiddleware.GetDeviceID(c), c.RealIP())
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "export failed", nil)
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=candidates_report.csv")
	c.Response().Header().Set("Content-Type", "text/csv")
	return c.Blob(http.StatusOK, "text/csv", csvData)
}
