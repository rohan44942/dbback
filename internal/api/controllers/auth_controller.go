package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AuthController handles authentication-related endpoints
type AuthController struct{}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type userProfile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type authResponse struct {
	User      userProfile `json:"user"`
	Token     string      `json:"token"`
	ExpiresAt int64       `json:"expiresAt"`
}

func NewAuthController() *AuthController {
	return &AuthController{}
}

// POST /auth/login
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}

	name := strings.Split(req.Email, "@")[0]
	writeAuthResponse(w, req.Email, name)
}

// POST /auth/register
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name, email, and password are required", http.StatusBadRequest)
		return
	}

	writeAuthResponse(w, req.Email, req.Name)
}

// POST /auth/logout
func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// POST /auth/refresh
func (c *AuthController) Refresh(w http.ResponseWriter, r *http.Request) {
	writeAuthResponse(w, "user@example.com", "User")
}

// GET /auth/me
func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userProfile{
		ID:        "dev-user",
		Email:     "user@example.com",
		Name:      "User",
		CreatedAt: time.Now().Format(time.RFC3339),
	})
}

func writeAuthResponse(w http.ResponseWriter, email, name string) {
	now := time.Now()
	resp := authResponse{
		User: userProfile{
			ID:        uuid.New().String(),
			Email:     email,
			Name:      name,
			CreatedAt: now.Format(time.RFC3339),
		},
		Token:     "dev-" + uuid.New().String(),
		ExpiresAt: now.Add(24 * time.Hour).UnixMilli(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
