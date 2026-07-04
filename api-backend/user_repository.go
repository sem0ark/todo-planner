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
func (r *UserRepository) Create(ctx context.Context, email, password string) (*User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var user User
	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, created_at`
	err = r.db.QueryRow(ctx, query, email, string(passwordHash)).Scan(&user.ID, &user.Email, &user.CreatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}

	return &user, nil
}

// FindByEmail retrieves a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	query := `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
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
	ErrDuplicateEmail = &DuplicateEmailError{}
)

type DuplicateEmailError struct{}

func (e *DuplicateEmailError) Error() string {
	return "email already exists"
}
