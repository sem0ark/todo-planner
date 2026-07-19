package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A registered client device
type Device struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`
	Platform     string     `json:"platform"` // desktop | mobile | web
	TokenHash    string     `json:"-"`
	RegisteredAt time.Time  `json:"registered_at"`
	LastSyncAt   *time.Time `json:"last_sync_at"`
}

type DeviceRepository struct {
	db *pgxpool.Pool
}

func NewDeviceRepository(db *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Create(ctx context.Context, userID int, platform string) (*Device, error) {
	token, err := generateDeviceToken()
	if err != nil {
		return nil, err
	}

	tokenHash := hashToken(token)
	var device Device

	err = r.db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, platform, registered_at
	`, userID, platform, tokenHash, time.Now()).Scan(
		&device.ID,
		&device.UserID,
		&device.Platform,
		&device.RegisteredAt,
	)
	if err != nil {
		return nil, err
	}

	return &device, nil
}

func generateDeviceToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
