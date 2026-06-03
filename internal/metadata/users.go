package metadata

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrUserExists = errors.New("user already exists")

func (s *Store) CreateUser(email, passwordHash, name string) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	id := uuid.New().String()
	row := s.db.QueryRow(
		`INSERT INTO users(id, email, password_hash, name)
		 VALUES($1, $2, $3, $4)
		 RETURNING id, email, password_hash, name, created_at`,
		id, email, passwordHash, name,
	)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
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
		`SELECT id, email, password_hash, name, created_at FROM users WHERE LOWER(email) = $1`, email)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, sql.ErrNoRows
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) GetUserByID(id string) (User, error) {
	row := s.db.QueryRow(
		`SELECT id, email, password_hash, name, created_at FROM users WHERE id = $1`, id)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, sql.ErrNoRows
		}
		return User{}, err
	}
	return u, nil
}
