package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	tokenauth "github.com/rohan44942/dbback/internal/auth"
	"github.com/rohan44942/dbback/internal/config"
	"github.com/rohan44942/dbback/internal/metadata"
)

type AuthController struct {
	Store  *metadata.Store
	Secret string
}

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

func NewAuthController(store *metadata.Store) *AuthController {
	return &AuthController{
		Store:  store,
		Secret: config.AuthSecret(),
	}
}

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

	user, err := c.Store.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}
	c.writeAuthResponse(w, user)
}

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
	if len(req.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := c.Store.CreateUser(req.Email, string(hash), strings.TrimSpace(req.Name))
	if err != nil {
		if errors.Is(err, metadata.ErrUserExists) {
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "registration failed", http.StatusInternalServerError)
		return
	}
	c.writeAuthResponse(w, user)
}

func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (c *AuthController) Refresh(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromBearer(r, c.Secret)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	user, err := c.Store.GetUserByID(claims.Subject)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}
	c.writeAuthResponse(w, user)
}

func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromBearer(r, c.Secret)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	user, err := c.Store.GetUserByID(claims.Subject)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserProfile(user))
}

func (c *AuthController) writeAuthResponse(w http.ResponseWriter, user metadata.User) {
	expiresAt := time.Now().Add(24 * time.Hour)
	token, err := tokenauth.GenerateToken(tokenauth.Claims{
		Subject: user.ID,
		Email:   user.Email,
		Name:    user.Name,
		Expires: expiresAt.Unix(),
	}, c.Secret)
	if err != nil {
		http.Error(w, "failed to create auth token", http.StatusInternalServerError)
		return
	}

	resp := authResponse{
		User:      toUserProfile(user),
		Token:     token,
		ExpiresAt: expiresAt.UnixMilli(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func toUserProfile(user metadata.User) userProfile {
	return userProfile{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}

func claimsFromBearer(r *http.Request, secret string) (tokenauth.Claims, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return tokenauth.Claims{}, tokenauth.ErrInvalidToken
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return tokenauth.ValidateToken(token, secret, time.Now())
}
