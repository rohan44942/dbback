package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tokenauth "github.com/rohan44942/dbback/internal/auth"
)

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	handler := RequireAuth("test-secret", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/api/backups", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuthAcceptsValidToken(t *testing.T) {
	token, err := tokenauth.GenerateToken(tokenauth.Claims{
		Subject: "user-1",
		Email:   "user@example.com",
		Name:    "User",
		Expires: time.Now().Add(time.Hour).Unix(),
	}, "test-secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	handler := RequireAuth("test-secret", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims.Subject != "user-1" {
			t.Fatalf("claims = %+v, ok = %v", claims, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/backups", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
