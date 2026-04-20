package totp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// GenerateSecret generates a new TOTP secret and returns the base32 secret,
// an otpauth:// URI, and a base64-encoded PNG QR code data URI.
func GenerateSecret() (secret string, qrData string, qrImageData string, err error) {
	issuer := os.Getenv("TOTP_ISSUER")
	if issuer == "" {
		issuer = os.Getenv("APP_NAME")
	}
	if issuer == "" {
		issuer = "Keystone"
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: "user@keystone",
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	img, err := key.Image(200, 200)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate QR image: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", "", fmt.Errorf("failed to encode QR image: %w", err)
	}

	qrImageData = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	return key.Secret(), key.URL(), qrImageData, nil
}

// VerifyCode verifies a 6-digit TOTP code against the given base32 secret.
func VerifyCode(secret string, code string) bool {
	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return false
	}
	return valid
}
