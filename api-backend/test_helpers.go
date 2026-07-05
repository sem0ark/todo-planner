package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration tests")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := ApplyMigrations(ctx, db, GetMigrations()); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	t.Cleanup(func() {
		cleanupTestDB(t, db)
		db.Close()
	})

	return db
}

func cleanupTestDB(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	tables := []string{
		"actual_blocks", "retroactive_edits", "day_events", "day_records",
		"schedule_overrides", "weekly_schedule", "snapshot_blocks",
		"template_snapshots", "planned_blocks", "day_templates",
		"template_groups", "block_categories", "change_log", "devices",
		"user_settings", "users",
	}

	for _, table := range tables {
		_, err := db.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Failed to truncate %s: %v", table, err)
		}
	}
}

func createTestUser(t *testing.T, db *pgxpool.Pool, username, password string) *User {
	t.Helper()

	repo := NewUserRepository(db)
	_, err := repo.Create(context.Background(), username, password)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Fetch full user with password_hash for tests that need it
	fullUser, err := repo.FindByUsername(context.Background(), username)
	if err != nil {
		t.Fatalf("Failed to fetch created user: %v", err)
	}

	return fullUser
}
