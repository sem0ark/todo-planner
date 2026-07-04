package main

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository handles all database operations for users
type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create registers a new user with hashed password
func (r *UserRepository) Create(ctx context.Context, username, password string) (*User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var user User
	query := `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id, username, created_at`
	err = r.db.QueryRow(ctx, query, username, string(passwordHash)).Scan(&user.ID, &user.Username, &user.CreatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, ErrDuplicateUsername
		}
		return nil, err
	}

	return &user, nil
}

// FindByUsername retrieves a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	query := `SELECT id, username, password_hash, created_at FROM users WHERE username = $1`
	err := r.db.QueryRow(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// VerifyPassword checks if password matches the user's stored hash
func (r *UserRepository) VerifyPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// Common errors
var (
	ErrDuplicateUsername = &DuplicateUsernameError{}
)

type DuplicateUsernameError struct{}

func (e *DuplicateUsernameError) Error() string {
	return "username already exists"
}
