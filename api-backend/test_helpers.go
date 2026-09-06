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
		"users",
		"user_settings",
		"devices",
		"change_log",
		"block_categories",
		"template_groups",
		"day_templates",
		"template_snapshots",
		"snapshot_blocks",
		"weekly_schedule",
		"schedule_overrides",
		"day_records",
		"day_events",
		"actual_blocks",
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

	repo := NewUserRepositoryEmptyDefault(db)
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

func createTestCategory(t *testing.T, db *pgxpool.Pool, userID int, name, color string) *BlockCategory {
	t.Helper()

	repo := NewCategoryRepository(db)
	category, err := repo.Create(context.Background(), CategoryInput{Name: name, Color: color}, userID)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	return category
}

func createTestTemplateGroup(t *testing.T, db *pgxpool.Pool, userID int, name string) *TemplateGroup {
	t.Helper()

	repo := NewTemplateGroupRepository(db)
	group, err := repo.Create(context.Background(), TemplateGroupInput{Name: name}, userID)
	if err != nil {
		t.Fatalf("Failed to create test template group: %v", err)
	}

	return group
}

func createTestDayTemplate(t *testing.T, db *pgxpool.Pool, userID int, name string, groupID *int) *DayTemplate {
	t.Helper()

	repo := NewDayTemplateRepository(db)
	template, err := repo.Create(context.Background(), DayTemplateInput{Name: name, TemplateGroupID: groupID}, userID)
	if err != nil {
		t.Fatalf("Failed to create test day template: %v", err)
	}

	return template
}
