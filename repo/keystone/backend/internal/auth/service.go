package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/keystone/backend/internal/db"
	appMiddleware "github.com/keystone/backend/internal/middleware"
	"github.com/keystone/backend/pkg/crypto"
	appTOTP "github.com/keystone/backend/pkg/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	maxFailedAttempts = 5
	lockDuration      = 15 * time.Minute
	jwtTTL            = 24 * time.Hour
)

// AccountLockedError is returned when the account is currently locked, carrying the unlock time.
type AccountLockedError struct {
	LockoutUntil time.Time
}

func (e *AccountLockedError) Error() string { return "account locked" }

// ErrAccountLocked is a sentinel kept for errors.Is compatibility.
var ErrAccountLocked = errors.New("account locked")

// ErrInvalidCredentials is returned on wrong password.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrMFARequired is returned when MFA verification is still needed.
var ErrMFARequired = errors.New("MFA required")

// ErrInvalidMFA is returned when the TOTP code is wrong.
var ErrInvalidMFA = errors.New("invalid MFA code")

// AuditLogger is a minimal interface for writing audit events without a circular import.
type AuditLogger interface {
	Log(actorID, action, resourceType, resourceID, deviceID, ip string, before, after interface{}) error
}

// Service implements business logic for authentication.
type Service struct {
	repo   *Repository
	audit  AuditLogger
}

// NewService creates a new auth Service.
func NewService(repo *Repository, audit AuditLogger) *Service {
	return &Service{repo: repo, audit: audit}
}

func generateJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "changeme-jwt-secret"
	}
	return []byte(s)
}

// Login authenticates a user and returns a JWT on success.
func (s *Service) Login(req LoginRequest, deviceID, ip string) (*LoginResponse, error) {
	user, err := s.repo.GetUserByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check lockout.
	if user.IsLocked && user.LockTime != nil {
		if time.Since(*user.LockTime) < lockDuration {
			_ = s.audit.Log(user.ID, "LOGIN_BLOCKED_LOCKED", "user", user.ID, deviceID, ip, nil, nil)
			return nil, &AccountLockedError{LockoutUntil: user.LockTime.Add(lockDuration)}
		}
		// Auto-unlock after 15 minutes.
		if err := s.repo.UpdateFailedAttempts(user.ID, 0, nil); err != nil {
			return nil, err
		}
		user.IsLocked = false
		user.FailedAttempts = 0
	}

	// Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		newAttempts := user.FailedAttempts + 1
		var lockTime *time.Time
		if newAttempts >= maxFailedAttempts {
			now := time.Now()
			lockTime = &now
		}
		_ = s.repo.UpdateFailedAttempts(user.ID, newAttempts, lockTime)
		_ = s.audit.Log(user.ID, "LOGIN_FAILED", "user", user.ID, deviceID, ip, nil, map[string]interface{}{"failedAttempts": newAttempts})
		return nil, ErrInvalidCredentials
	}

	// Successful password check – verify TOTP if enabled.
	if user.MFAEnabled {
		if req.TOTPCode == "" {
			return nil, ErrMFARequired
		}
		decryptedSecret, err := crypto.DecryptAES(user.MFASecret, crypto.GetAESKey())
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt MFA secret: %w", err)
		}
		if !appTOTP.VerifyCode(decryptedSecret, req.TOTPCode) {
			_ = s.audit.Log(user.ID, "LOGIN_MFA_FAILED", "user", user.ID, deviceID, ip, nil, nil)
			return nil, ErrInvalidMFA
		}
	}

	// Reset failed attempts on success.
	_ = s.repo.UpdateFailedAttempts(user.ID, 0, nil)

	// Generate JWT.
	menuPerms := s.GetMenuPermissions(user.Role)
	claims := &appMiddleware.UserClaims{
		UserID:         user.ID,
		Username:       user.Username,
		Email:          user.Email,
		Role:           user.Role,
		Permissions:    menuPerms,
		SiteID:         user.SiteID,
		OrganizationID: user.OrganizationID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        generateJTI(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecret())
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT: %w", err)
	}

	// Persist session.
	session := &db.Session{
		UserID:    user.ID,
		Token:     tokenStr,
		DeviceID:  deviceID,
		IPAddress: ip,
		ExpiresAt: time.Now().Add(jwtTTL),
	}
	_ = s.repo.CreateSession(session)

	_ = s.audit.Log(user.ID, "LOGIN_SUCCESS", "user", user.ID, deviceID, ip, nil, nil)

	return &LoginResponse{
		Token:           tokenStr,
		Role:            user.Role,
		MenuPermissions: menuPerms,
		User: UserDTO{
			ID:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			Role:       user.Role,
			MFAEnabled: user.MFAEnabled,
		},
	}, nil
}

// Logout invalidates the given JWT token and writes an audit record.
func (s *Service) Logout(token, actorID, deviceID, ip string) error {
	if err := s.repo.InvalidateSession(token); err != nil {
		return err
	}
	_ = s.audit.Log(actorID, "LOGOUT", "user", actorID, deviceID, ip, nil, nil)
	return nil
}

// GetMe returns the public profile of a user.
func (s *Service) GetMe(userID string) (*UserDTO, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	return &UserDTO{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Role:       user.Role,
		MFAEnabled: user.MFAEnabled,
	}, nil
}

// SetupMFA generates a TOTP secret, encrypts it, and stores it in the user record.
func (s *Service) SetupMFA(userID string) (*MFASetupResponse, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	secret, qrData, qrImageData, err := appTOTP.GenerateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	encrypted, err := crypto.EncryptAES(secret, crypto.GetAESKey())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}

	user.MFASecret = encrypted
	// MFA is not yet enabled – user must verify first.
	if err := s.repo.UpdateUser(user); err != nil {
		return nil, err
	}

	return &MFASetupResponse{Secret: secret, QRData: qrData, QRImageData: qrImageData}, nil
}

// VerifyMFA confirms a TOTP code and enables MFA on the user account.
func (s *Service) VerifyMFA(userID, code string) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user.MFASecret == "" {
		return errors.New("MFA not set up")
	}

	decryptedSecret, err := crypto.DecryptAES(user.MFASecret, crypto.GetAESKey())
	if err != nil {
		return fmt.Errorf("failed to decrypt MFA secret: %w", err)
	}

	if !appTOTP.VerifyCode(decryptedSecret, code) {
		return ErrInvalidMFA
	}

	user.MFAEnabled = true
	return s.repo.UpdateUser(user)
}

// GetMenuPermissions returns the set of frontend routes and action permissions for a given role.
func (s *Service) GetMenuPermissions(role string) []string {
	switch role {
	case "ADMIN":
		return []string{
			"/candidates", "/candidates/new",
			"/listings",
			"/parts", "/parts/import",
			"/audit-logs",
			"/reports",
			"/search",
			"/users",
			"*",
		}
	case "INTAKE_SPECIALIST":
		return []string{
			"/candidates", "/candidates/new", "/listings",
			"documents:download",
		}
	case "REVIEWER":
		return []string{
			"/candidates", "/listings",
			"documents:download",
		}
	case "INVENTORY_CLERK":
		return []string{
			"/listings", "/parts", "/parts/import",
			"parts:export",
		}
	case "AUDITOR":
		return []string{
			"/audit-logs", "/candidates", "/listings", "/parts",
			"reports:view", "reports:export", "parts:export",
		}
	default:
		return []string{}
	}
}

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CreateUser creates a new user (admin function).
func (s *Service) CreateUser(req CreateUserRequest) (*UserDTO, error) {
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &db.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
		Permissions:  req.Permissions,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}

	return &UserDTO{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Role:       user.Role,
		MFAEnabled: user.MFAEnabled,
	}, nil
}

// ListUsers returns all users (admin function).
func (s *Service) ListUsers() ([]UserDTO, error) {
	users, err := s.repo.ListUsers()
	if err != nil {
		return nil, err
	}
	dtos := make([]UserDTO, len(users))
	for i, u := range users {
		dtos[i] = UserDTO{
			ID:         u.ID,
			Username:   u.Username,
			Email:      u.Email,
			Role:       u.Role,
			MFAEnabled: u.MFAEnabled,
		}
	}
	return dtos, nil
}
