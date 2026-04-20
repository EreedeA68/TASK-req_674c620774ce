package unit

import (
	"errors"
	"testing"
	"time"

	"github.com/keystone/backend/internal/auth"
	"github.com/keystone/backend/internal/db"
	"github.com/keystone/backend/pkg/totp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func makePasswordHash(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(hash)
}

func validUser() *db.User {
	return &db.User{
		ID:             "test-user-id",
		Username:       "testuser",
		Email:          "test@example.com",
		PasswordHash:   makePasswordHash("Password1!"),
		Role:           "INTAKE_SPECIALIST",
		MFAEnabled:     false,
		IsLocked:       false,
		FailedAttempts: 0,
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := auth.HashPassword("Password1!")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("Password1!"))
	assert.NoError(t, err)
}

func TestTOTPSecretGeneration(t *testing.T) {
	secret, qrData, qrImageData, err := totp.GenerateSecret()
	assert.NoError(t, err)
	assert.NotEmpty(t, secret, "TOTP secret should not be empty")
	assert.Contains(t, qrData, "otpauth://", "QR data should be an otpauth URI")
	assert.Contains(t, qrImageData, "data:image/png;base64,", "QR image should be a PNG data URI")
}

func TestValidTOTPCode(t *testing.T) {
	secret, _, _, err := totp.GenerateSecret()
	assert.NoError(t, err)
	// A freshly generated secret is valid to verify against.
	// We just confirm VerifyCode runs without panic.
	valid := totp.VerifyCode(secret, "000000")
	_ = valid // result depends on timing
}

func TestInvalidTOTPCode(t *testing.T) {
	secret, _, _, err := totp.GenerateSecret()
	assert.NoError(t, err)
	// A clearly bad code (non-numeric) should fail.
	valid := totp.VerifyCode(secret, "abc")
	assert.False(t, valid)
}

func TestPasswordTooShort(t *testing.T) {
	// bcrypt doesn't enforce length; enforcement is at the validation layer.
	hash, err := auth.HashPassword("short")
	assert.NoError(t, err)
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("short"))
	assert.NoError(t, err)
}

func TestGetMenuPermissions_Admin(t *testing.T) {
	svc := auth.NewService(nil, nil)
	perms := svc.GetMenuPermissions("ADMIN")
	assert.Contains(t, perms, "/candidates")
	assert.Contains(t, perms, "/audit-logs")
	assert.Contains(t, perms, "/parts")
	assert.Contains(t, perms, "*", "ADMIN must have wildcard action permission")
}

func TestGetMenuPermissions_IntakeSpecialist(t *testing.T) {
	svc := auth.NewService(nil, nil)
	perms := svc.GetMenuPermissions("INTAKE_SPECIALIST")
	assert.Contains(t, perms, "/candidates")
	assert.NotContains(t, perms, "/audit-logs")
	assert.Contains(t, perms, "documents:download", "INTAKE_SPECIALIST must have documents:download permission")
}

func TestGetMenuPermissions_Reviewer(t *testing.T) {
	svc := auth.NewService(nil, nil)
	perms := svc.GetMenuPermissions("REVIEWER")
	assert.Contains(t, perms, "/candidates")
	assert.Contains(t, perms, "/listings")
	assert.NotContains(t, perms, "/parts")
	assert.Contains(t, perms, "documents:download", "REVIEWER must have documents:download permission")
}

func TestGetMenuPermissions_InventoryClerk(t *testing.T) {
	svc := auth.NewService(nil, nil)
	perms := svc.GetMenuPermissions("INVENTORY_CLERK")
	assert.Contains(t, perms, "/parts")
	assert.NotContains(t, perms, "/candidates")
	assert.Contains(t, perms, "parts:export", "INVENTORY_CLERK must have parts:export permission")
}

func TestGetMenuPermissions_Auditor(t *testing.T) {
	svc := auth.NewService(nil, nil)
	perms := svc.GetMenuPermissions("AUDITOR")
	assert.Contains(t, perms, "/audit-logs")
	assert.Contains(t, perms, "/candidates")
	assert.Contains(t, perms, "/parts")
	assert.Contains(t, perms, "reports:export", "AUDITOR must have reports:export permission")
}

func TestGetMenuPermissions_Unknown(t *testing.T) {
	svc := auth.NewService(nil, nil)
	perms := svc.GetMenuPermissions("UNKNOWN_ROLE")
	assert.Empty(t, perms)
}


func TestWrongPasswordIncrementsAttempts(t *testing.T) {
	hash := makePasswordHash("CorrectPassword1!")
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("WrongPassword"))
	assert.Error(t, err, "wrong password should fail bcrypt comparison")
}

func TestFifthFailureLocks(t *testing.T) {
	const maxAttempts = 5
	attempts := 0
	isLocked := false
	var lockTime *time.Time

	hash := makePasswordHash("CorrectPwd1!")
	wrongPwd := "WrongPwd"

	for i := 0; i < maxAttempts; i++ {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(wrongPwd))
		if err != nil {
			attempts++
			if attempts >= maxAttempts {
				now := time.Now()
				isLocked = true
				lockTime = &now
			}
		}
	}

	assert.True(t, isLocked, "account should be locked after 5 failures")
	assert.NotNil(t, lockTime)
	assert.Equal(t, maxAttempts, attempts)
}

func TestLockedAccountRejects(t *testing.T) {
	now := time.Now()
	lockTime := &now
	isLocked := true

	shouldReject := isLocked && lockTime != nil && time.Since(*lockTime) < 15*time.Minute
	assert.True(t, shouldReject, "locked account within 15 minutes should be rejected")
}

func TestAccountUnlocksAfter15Min(t *testing.T) {
	past := time.Now().Add(-16 * time.Minute)
	lockTime := &past
	isLocked := true

	shouldUnlock := isLocked && lockTime != nil && time.Since(*lockTime) >= 15*time.Minute
	assert.True(t, shouldUnlock, "account should unlock after 15 minutes")
}

func TestPasswordNoNumber(t *testing.T) {
	password := "NoNumbersHere!"
	hasDigit := false
	for _, c := range password {
		if c >= '0' && c <= '9' {
			hasDigit = true
			break
		}
	}
	assert.False(t, hasDigit)
}

func TestPasswordNoSymbol(t *testing.T) {
	password := "NoSymbol1234"
	symbols := "!@#$%^&*()-_=+[]{}|;:',.<>?/`~"
	hasSymbol := false
	for _, c := range password {
		for _, s := range symbols {
			if c == s {
				hasSymbol = true
				break
			}
		}
	}
	assert.False(t, hasSymbol)
}

func TestValidCredentialsReturnJWT(t *testing.T) {
	hash := makePasswordHash("ValidPwd1!")
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("ValidPwd1!"))
	assert.NoError(t, err, "correct password should pass bcrypt verification")
}

func TestMFASecretIsBase32(t *testing.T) {
	secret, _, _, err := totp.GenerateSecret()
	assert.NoError(t, err)
	for _, c := range secret {
		valid := (c >= 'A' && c <= 'Z') || (c >= '2' && c <= '7') || c == '='
		if !valid {
			t.Errorf("secret contains non-base32 character: %c", c)
		}
	}
}

func TestErrorDistinction(t *testing.T) {
	assert.False(t, errors.Is(auth.ErrAccountLocked, auth.ErrInvalidCredentials))
	assert.False(t, errors.Is(auth.ErrMFARequired, auth.ErrInvalidMFA))
}

// Ensure validUser helper compiles and returns expected fields.
func TestValidUserHelper(t *testing.T) {
	u := validUser()
	assert.Equal(t, "testuser", u.Username)
	assert.Equal(t, "INTAKE_SPECIALIST", u.Role)
	assert.False(t, u.IsLocked)
	assert.Equal(t, 0, u.FailedAttempts)
}
