package metadata

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrUserExists = errors.New("user already exists")

const (
	AuthProviderLocal  = "local"
	AuthProviderGoogle = "google"
)

func (s *Store) CreateUser(email, passwordHash, name string) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	id := uuid.New().String()
	row := s.db.QueryRow(
		`INSERT INTO users(id, email, password_hash, name, auth_provider)
		 VALUES($1, $2, $3, $4, $5)
		 RETURNING id, email, COALESCE(password_hash, ''), name, COALESCE(google_id, ''), auth_provider, created_at`,
		id, email, passwordHash, name, AuthProviderLocal,
	)
	return scanUser(row)
}

func (s *Store) CreateGoogleUser(email, name, googleID string) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	id := uuid.New().String()
	row := s.db.QueryRow(
		`INSERT INTO users(id, email, password_hash, name, google_id, auth_provider)
		 VALUES($1, $2, NULL, $3, $4, $5)
		 RETURNING id, email, COALESCE(password_hash, ''), name, COALESCE(google_id, ''), auth_provider, created_at`,
		id, email, name, googleID, AuthProviderGoogle,
	)
	u, err := scanUser(row)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return User{}, ErrUserExists
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) GetUserByEmail(email string) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	row := s.db.QueryRow(
		`SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(google_id, ''), COALESCE(auth_provider, 'local'), created_at
		 FROM users WHERE LOWER(email) = $1`, email)
	return scanUser(row)
}

func (s *Store) GetUserByID(id string) (User, error) {
	row := s.db.QueryRow(
		`SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(google_id, ''), COALESCE(auth_provider, 'local'), created_at
		 FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (s *Store) GetUserByGoogleID(googleID string) (User, error) {
	googleID = strings.TrimSpace(googleID)
	if googleID == "" {
		return User{}, sql.ErrNoRows
	}
	row := s.db.QueryRow(
		`SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(google_id, ''), COALESCE(auth_provider, 'local'), created_at
		 FROM users WHERE google_id = $1`, googleID)
	return scanUser(row)
}

func (s *Store) LinkGoogleID(userID, googleID string) error {
	_, err := s.db.Exec(
		`UPDATE users SET google_id = $1 WHERE id = $2 AND (google_id IS NULL OR google_id = '')`,
		googleID, userID,
	)
	return err
}

func scanUser(row *sql.Row) (User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.GoogleID, &u.AuthProvider, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, sql.ErrNoRows
		}
		return User{}, err
	}
	return u, nil
}
