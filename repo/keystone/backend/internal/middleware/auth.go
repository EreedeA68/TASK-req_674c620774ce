package middleware

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	internalDB "github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

// UserClaims holds the JWT payload for authenticated users.
type UserClaims struct {
	UserID         string   `json:"userId"`
	Username       string   `json:"username"`
	Email          string   `json:"email"`
	Role           string   `json:"role"`
	Permissions    []string `json:"permissions"`
	SiteID         string   `json:"siteId,omitempty"`
	OrganizationID string   `json:"organizationId,omitempty"`
	jwt.RegisteredClaims
}

const userContextKey = "user"

func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "changeme-jwt-secret"
	}
	return []byte(secret)
}

// ValidateJWT is an Echo middleware that validates a Bearer JWT token (signature only).
func ValidateJWT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, claims, err := parseJWT(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"data":         nil,
				"errorMessage": err.Error(),
				"code":         http.StatusUnauthorized,
				"details":      nil,
			})
		}
		c.Set(userContextKey, claims)
		return next(c)
	}
}

// ValidateJWTWithDB returns middleware that validates a JWT and checks the session is active in the DB.
func ValidateJWTWithDB(database *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr, claims, err := parseJWT(c)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"data":         nil,
					"errorMessage": err.Error(),
					"code":         http.StatusUnauthorized,
					"details":      nil,
				})
			}

			var count int64
			if dbErr := database.Model(&internalDB.Session{}).
				Where("token = ? AND invalidated = false AND expires_at > NOW()", tokenStr).
				Count(&count).Error; dbErr != nil || count == 0 {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"data":         nil,
					"errorMessage": "session expired or invalidated",
					"code":         http.StatusUnauthorized,
					"details":      nil,
				})
			}

			c.Set(userContextKey, claims)
			return next(c)
		}
	}
}

// parseJWT extracts and validates the JWT from the Authorization header.
func parseJWT(c echo.Context) (string, *UserClaims, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", nil, errors.New("missing or invalid authorization header")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &UserClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		return "", nil, errors.New("invalid or expired token")
	}
	return tokenStr, claims, nil
}

// RequireRole returns middleware that restricts access to the listed roles.
func RequireRole(roles ...string) echo.MiddlewareFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := GetUserFromContext(c)
			if user == nil || !allowed[user.Role] {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"data":         nil,
					"errorMessage": "insufficient permissions",
					"code":         http.StatusForbidden,
					"details":      nil,
				})
			}
			return next(c)
		}
	}
}

// RequirePermission returns middleware that checks whether a specific permission is present.
func RequirePermission(resource, action string) echo.MiddlewareFunc {
	required := resource + ":" + action
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := GetUserFromContext(c)
			if user == nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"data":         nil,
					"errorMessage": "unauthorized",
					"code":         http.StatusUnauthorized,
					"details":      nil,
				})
			}
			for _, p := range user.Permissions {
				if p == required || p == "*" {
					return next(c)
				}
			}
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"data":         nil,
				"errorMessage": "insufficient permissions",
				"code":         http.StatusForbidden,
				"details":      nil,
			})
		}
	}
}

// GetUserFromContext retrieves the authenticated user claims from the Echo context.
func GetUserFromContext(c echo.Context) *UserClaims {
	val := c.Get(userContextKey)
	if val == nil {
		return nil
	}
	claims, _ := val.(*UserClaims)
	return claims
}

// GetDeviceID extracts a device identifier from request headers.
func GetDeviceID(c echo.Context) string {
	if id := c.Request().Header.Get("X-Device-ID"); id != "" {
		return id
	}
	if id := c.Request().Header.Get("X-Device-Id"); id != "" {
		return id
	}
	return c.Request().UserAgent()
}
