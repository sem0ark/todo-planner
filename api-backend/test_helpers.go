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
	transaction, err := db.Begin(context.Background())
	if err != nil {
		t.Fatalf("Failed to start test configuration transaction: %v", err)
	}
	testConfiguration := DefaultUserConfiguration{
		Version: 1,
		Settings: DefaultSettingsConfiguration{
			DayRangeStartTime: "04:00:00",
			DayRangeEndTime:   "23:00:00",
		},
		WeeklySchedule: []DefaultWeeklyScheduleEntry{
			{DayOfWeek: 0}, {DayOfWeek: 1}, {DayOfWeek: 2}, {DayOfWeek: 3},
			{DayOfWeek: 4}, {DayOfWeek: 5}, {DayOfWeek: 6},
		},
	}
	if err := NewBackupRepository(nil).ImportConfiguration(context.Background(), transaction, fullUser.ID, testConfiguration); err != nil {
		transaction.Rollback(context.Background())
		t.Fatalf("Failed to create test user configuration: %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("Failed to commit test user configuration: %v", err)
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

func mustScheduleTime(value string) ScheduleTime {
	parsedTime, err := parseScheduleTime(value)
	if err != nil {
		panic(err)
	}
	return parsedTime
}

func mustCalendarDate(value string) CalendarDate {
	parsedDate, err := parseCalendarDate(value)
	if err != nil {
		panic(err)
	}
	return parsedDate
}
