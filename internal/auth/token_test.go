package auth

import (
	"errors"
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	now := time.Unix(1000, 0)
	token, err := GenerateToken(Claims{
		Subject: "user-1",
		Email:   "user@example.com",
		Name:    "User",
		Expires: now.Add(time.Hour).Unix(),
	}, "test-secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ValidateToken(token, "test-secret", now)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.Subject != "user-1" || claims.Email != "user@example.com" || claims.Name != "User" {
		t.Fatalf("ValidateToken() claims = %+v", claims)
	}
}

func TestValidateTokenRejectsWrongSecret(t *testing.T) {
	now := time.Unix(1000, 0)
	token, err := GenerateToken(Claims{
		Subject: "user-1",
		Email:   "user@example.com",
		Name:    "User",
		Expires: now.Add(time.Hour).Unix(),
	}, "test-secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = ValidateToken(token, "wrong-secret", now)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ValidateToken() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestValidateTokenRejectsExpiredToken(t *testing.T) {
	now := time.Unix(1000, 0)
	token, err := GenerateToken(Claims{
		Subject: "user-1",
		Email:   "user@example.com",
		Name:    "User",
		Expires: now.Add(-time.Second).Unix(),
	}, "test-secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = ValidateToken(token, "test-secret", now)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("ValidateToken() error = %v, want %v", err, ErrExpiredToken)
	}
}
