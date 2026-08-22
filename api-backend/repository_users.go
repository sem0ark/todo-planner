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

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var user User
	query := `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id, username, created_at`
	err = tx.QueryRow(ctx, query, username, string(passwordHash)).Scan(&user.ID, &user.Username, &user.CreatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, ErrDuplicateUsername
		}
		return nil, err
	}

	if err := createDefaultUserConfiguration(ctx, tx, user.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
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

// DeleteAccount hard-deletes a user after password verification
func (r *UserRepository) DeleteAccount(ctx context.Context, userID int, password string) error {
	var user User
	query := `SELECT id, username, password_hash, created_at FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return err
	}

	if !r.VerifyPassword(&user, password) {
		return ErrInvalidPassword
	}

	deleteQuery := `DELETE FROM users WHERE id = $1`
	_, err = r.db.Exec(ctx, deleteQuery, userID)
	return err
}

// Common errors
var (
	ErrDuplicateUsername = &DuplicateUsernameError{}
	ErrInvalidPassword   = &InvalidPasswordError{}
)

type DuplicateUsernameError struct{}

func (e *DuplicateUsernameError) Error() string {
	return "username already exists"
}

type InvalidPasswordError struct{}

func (e *InvalidPasswordError) Error() string {
	return "invalid password"
}
