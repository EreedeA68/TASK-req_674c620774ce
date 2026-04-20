package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	users, err := h.svc.ListUsers()
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, "failed to fetch users", nil)
	}
	return respond(c, http.StatusOK, map[string]interface{}{"users": users}, "", nil)
}

func (h *AdminHandler) CreateUser(c echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request", nil)
	}
	if err := validatePasswordComplexity(req.Password); err != nil {
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}

	dto, err := h.svc.CreateUser(req)
	if err != nil {
		return respond(c, http.StatusConflict, nil, "username or email already exists", nil)
	}
	return respond(c, http.StatusCreated, dto, "", nil)
}
