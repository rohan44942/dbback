package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const tokenVersion = "v1"

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Expires int64  `json:"exp"`
}

func GenerateToken(claims Claims, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("auth secret is required")
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := sign(encodedPayload, secret)
	return fmt.Sprintf("%s.%s.%s", tokenVersion, encodedPayload, signature), nil
}

func ValidateToken(token, secret string, now time.Time) (Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return Claims{}, errors.New("auth secret is required")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != tokenVersion {
		return Claims{}, ErrInvalidToken
	}

	expectedSignature := sign(parts[1], secret)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.Expires <= now.Unix() {
		return Claims{}, ErrExpiredToken
	}

	return claims, nil
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
