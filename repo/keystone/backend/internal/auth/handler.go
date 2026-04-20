package auth

import (
	"errors"
	"net/http"
	"strings"
	"unicode"

	"github.com/labstack/echo/v4"

	appMiddleware "github.com/keystone/backend/internal/middleware"
)

// Handler handles HTTP requests for auth endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new auth Handler.
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

// Login handles POST /api/auth/login.
func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	deviceID := appMiddleware.GetDeviceID(c)
	ip := c.RealIP()

	resp, err := h.svc.Login(req, deviceID, ip)
	if err != nil {
		switch {
		case errors.As(err, new(*AccountLockedError)):
			var lockedErr *AccountLockedError
			errors.As(err, &lockedErr)
			return respond(c, http.StatusLocked, map[string]interface{}{
				"lockoutUntil": lockedErr.LockoutUntil.UTC().Format("2006-01-02T15:04:05Z"),
			}, "account is locked – try again later", nil)
		case errors.Is(err, ErrInvalidCredentials):
			return respond(c, http.StatusUnauthorized, nil, "invalid username or password", nil)
		case errors.Is(err, ErrMFARequired):
			return respond(c, http.StatusUnauthorized, nil, "MFA code required", "mfa_required")
		case errors.Is(err, ErrInvalidMFA):
			return respond(c, http.StatusUnauthorized, nil, "invalid MFA code", nil)
		default:
			return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
		}
	}

	return respond(c, http.StatusOK, resp, "", nil)
}

// Logout handles POST /api/auth/logout.
func (h *Handler) Logout(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return respond(c, http.StatusBadRequest, nil, "no token provided", nil)
	}

	actorID := ""
	if u := appMiddleware.GetUserFromContext(c); u != nil {
		actorID = u.UserID
	}

	if err := h.svc.Logout(token, actorID, appMiddleware.GetDeviceID(c), c.RealIP()); err != nil {
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusOK, map[string]string{"message": "logged out"}, "", nil)
}

// Me handles GET /api/auth/me.
func (h *Handler) Me(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	dto, err := h.svc.GetMe(user.UserID)
	if err != nil {
		return respond(c, http.StatusNotFound, nil, "user not found", nil)
	}

	return respond(c, http.StatusOK, dto, "", nil)
}

// SetupMFA handles POST /api/auth/mfa/setup.
func (h *Handler) SetupMFA(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	resp, err := h.svc.SetupMFA(user.UserID)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusOK, resp, "", nil)
}

// VerifyMFA handles POST /api/auth/mfa/verify.
func (h *Handler) VerifyMFA(c echo.Context) error {
	user := appMiddleware.GetUserFromContext(c)
	if user == nil {
		return respond(c, http.StatusUnauthorized, nil, "unauthorized", nil)
	}

	var req VerifyMFARequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}

	if err := h.svc.VerifyMFA(user.UserID, req.Code); err != nil {
		if errors.Is(err, ErrInvalidMFA) {
			return respond(c, http.StatusUnauthorized, nil, "invalid MFA code", nil)
		}
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusOK, map[string]string{"message": "MFA enabled successfully"}, "", nil)
}

// validatePasswordComplexity returns an error if the password does not meet complexity requirements.
func validatePasswordComplexity(pw string) error {
	if len(pw) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	var hasDigit, hasSymbol bool
	for _, ch := range pw {
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) {
			hasSymbol = true
		}
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	if !hasSymbol {
		return errors.New("password must contain at least one special character")
	}
	return nil
}

// CreateUser handles POST /api/users (admin only).
func (h *Handler) CreateUser(c echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return respond(c, http.StatusBadRequest, nil, "invalid request body", nil)
	}
	if err := validatePasswordComplexity(req.Password); err != nil {
		return respond(c, http.StatusBadRequest, nil, err.Error(), nil)
	}

	dto, err := h.svc.CreateUser(req)
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}

	return respond(c, http.StatusCreated, dto, "", nil)
}

// ListUsers handles GET /api/users (admin only).
func (h *Handler) ListUsers(c echo.Context) error {
	users, err := h.svc.ListUsers()
	if err != nil {
		return respond(c, http.StatusInternalServerError, nil, err.Error(), nil)
	}
	return respond(c, http.StatusOK, users, "", nil)
}
