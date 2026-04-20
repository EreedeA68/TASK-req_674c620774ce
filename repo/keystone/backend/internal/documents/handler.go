package documents

import (
	"net/http"

	"github.com/labstack/echo/v4"
	appMiddleware "github.com/keystone/backend/internal/middleware"
)

// Handler handles HTTP requests for document endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new documents Handler.
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

// Download handles GET /api/documents/:id/download.
func (h *Handler) Download(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	documentID := c.Param("id")
	deviceID := appMiddleware.GetDeviceID(c)

	data, fileName, err := h.svc.DownloadDocument(documentID, user.UserID, deviceID, c.RealIP())
	if err != nil {
		switch err.Error() {
		case "download not permitted":
			return respond(c, http.StatusForbidden, nil, err.Error(), nil)
		case "document not found":
			return respond(c, http.StatusNotFound, nil, err.Error(), nil)
		case "file not found on disk":
			return respond(c, http.StatusNotFound, nil, err.Error(), nil)
		case "file integrity check failed: hash mismatch":
			return respond(c, http.StatusInternalServerError, nil, "file integrity check failed", nil)
		default:
			return respond(c, http.StatusInternalServerError, nil, "download failed", nil)
		}
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename="+fileName)
	return c.Blob(http.StatusOK, "application/octet-stream", data)
}
