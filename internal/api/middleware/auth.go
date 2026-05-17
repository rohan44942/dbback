package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	tokenauth "github.com/rohan44942/dbback/internal/auth"
)

type contextKey string

const claimsContextKey contextKey = "authClaims"

func RequireAuth(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeAuthError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := tokenauth.ValidateToken(token, secret, time.Now())
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

func ClaimsFromContext(ctx context.Context) (tokenauth.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(tokenauth.Claims)
	return claims, ok
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
