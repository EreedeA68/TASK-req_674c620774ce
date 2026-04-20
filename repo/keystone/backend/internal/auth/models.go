package auth

import "encoding/json"

// LoginRequest is the payload for the login endpoint.
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	TOTPCode string `json:"totpCode"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	Token           string      `json:"token"`
	Role            string      `json:"role"`
	MenuPermissions []string    `json:"menuPermissions"`
	User            UserDTO     `json:"user"`
}

// UserDTO is the public representation of a user.
type UserDTO struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	MFAEnabled bool   `json:"mfaEnabled"`
}

// MFASetupResponse is returned when a user requests MFA setup.
type MFASetupResponse struct {
	Secret      string `json:"secret"`
	QRData      string `json:"qrData"`
	QRImageData string `json:"qrImageData"`
}

// VerifyMFARequest is the payload for verifying a TOTP code.
type VerifyMFARequest struct {
	Code string `json:"code" validate:"required"`
}

// ChangePasswordRequest is the payload for changing a password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

// CreateUserRequest is the admin payload for creating a new user.
type CreateUserRequest struct {
	Username    string          `json:"username" validate:"required"`
	Email       string          `json:"email" validate:"required,email"`
	Password    string          `json:"password" validate:"required,min=8"`
	Role        string          `json:"role" validate:"required"`
	Permissions json.RawMessage `json:"permissions"`
}
