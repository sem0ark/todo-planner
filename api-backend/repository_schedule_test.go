package main

import (
	"context"
	"testing"
	"time"
)

func TestScheduleRepository_GetWeeklySchedule(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	schedule, err := repo.GetWeeklySchedule(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("GetWeeklySchedule failed: %v", err)
	}
	if len(schedule) != 7 {
		t.Errorf("Expected 7 days, got %d", len(schedule))
	}
	for i := 0; i < 7; i++ {
		if schedule[i].DayOfWeek != i {
			t.Errorf("Expected day %d at index %d, got %d", i, i, schedule[i].DayOfWeek)
		}
		if schedule[i].DayTemplateID != nil {
			t.Errorf("Expected nil template ID for day %d, got %v", i, *schedule[i].DayTemplateID)
		}
	}
}

func TestScheduleRepository_ReplaceWeeklySchedule(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	template1 := createTestDayTemplate(t, db, user.ID, "Workday", nil)
	template2 := createTestDayTemplate(t, db, user.ID, "Weekend", nil)
	ctx := context.Background()

	entries := []WeeklyScheduleEntry{
		{DayOfWeek: 0, DayTemplateID: &template1.ID},
		{DayOfWeek: 1, DayTemplateID: &template2.ID},
		{DayOfWeek: 2, DayTemplateID: nil},
		{DayOfWeek: 3, DayTemplateID: &template1.ID},
		{DayOfWeek: 4, DayTemplateID: &template1.ID},
		{DayOfWeek: 5, DayTemplateID: nil},
		{DayOfWeek: 6, DayTemplateID: nil},
	}

	// Act
	updated, err := repo.ReplaceWeeklySchedule(ctx, user.ID, entries)

	// Assert
	if err != nil {
		t.Fatalf("ReplaceWeeklySchedule failed: %v", err)
	}
	if len(updated) != 7 {
		t.Errorf("Expected 7 days, got %d", len(updated))
	}
	if updated[0].DayTemplateID == nil || *updated[0].DayTemplateID != template1.ID {
		t.Errorf("Day 0 should have template ID %d", template1.ID)
	}
	if updated[1].DayTemplateID == nil || *updated[1].DayTemplateID != template2.ID {
		t.Errorf("Day 1 should have template ID %d", template2.ID)
	}
	if updated[2].DayTemplateID != nil {
		t.Error("Day 2 should have nil template ID")
	}
}

func TestScheduleRepository_ReplaceWeeklySchedule_NotEnoughDays(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	entries := []WeeklyScheduleEntry{
		{DayOfWeek: 0, DayTemplateID: nil},
		{DayOfWeek: 1, DayTemplateID: nil},
	}

	// Act
	_, err := repo.ReplaceWeeklySchedule(ctx, user.ID, entries)

	// Assert
	if err == nil {
		t.Error("Expected error for not enough days")
	}
}

func TestScheduleRepository_ReplaceWeeklySchedule_DuplicateDays(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	entries := []WeeklyScheduleEntry{
		{DayOfWeek: 0, DayTemplateID: nil},
		{DayOfWeek: 0, DayTemplateID: nil},
		{DayOfWeek: 2, DayTemplateID: nil},
		{DayOfWeek: 3, DayTemplateID: nil},
		{DayOfWeek: 4, DayTemplateID: nil},
		{DayOfWeek: 5, DayTemplateID: nil},
		{DayOfWeek: 6, DayTemplateID: nil},
	}

	// Act
	_, err := repo.ReplaceWeeklySchedule(ctx, user.ID, entries)

	// Assert
	if err == nil {
		t.Error("Expected error for duplicate days")
	}
}

func TestScheduleRepository_ReplaceWeeklySchedule_InvalidDayOfWeek(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	entries := []WeeklyScheduleEntry{
		{DayOfWeek: 0, DayTemplateID: nil},
		{DayOfWeek: 1, DayTemplateID: nil},
		{DayOfWeek: 2, DayTemplateID: nil},
		{DayOfWeek: 3, DayTemplateID: nil},
		{DayOfWeek: 4, DayTemplateID: nil},
		{DayOfWeek: 5, DayTemplateID: nil},
		{DayOfWeek: 7, DayTemplateID: nil}, // Invalid
	}

	// Act
	_, err := repo.ReplaceWeeklySchedule(ctx, user.ID, entries)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid day_of_week")
	}
}

func TestScheduleRepository_GetFutureOverrides(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	template := createTestDayTemplate(t, db, user.ID, "Holiday", nil)
	ctx := context.Background()

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	dayAfter := time.Now().AddDate(0, 0, 2).Format("2006-01-02")

	repo.SetOverride(ctx, user.ID, tomorrow, &template.ID)
	repo.SetOverride(ctx, user.ID, dayAfter, nil)

	// Act
	overrides, err := repo.GetFutureOverrides(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("GetFutureOverrides failed: %v", err)
	}
	if len(overrides) != 1 {
		t.Errorf("Expected 1 override (nil template ID is deleted), got %d", len(overrides))
	}
	if overrides[0].CalendarDate != tomorrow {
		t.Errorf("Expected override for %s, got %s", tomorrow, overrides[0].CalendarDate)
	}
}

func TestScheduleRepository_SetOverride_Create(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	template := createTestDayTemplate(t, db, user.ID, "Holiday", nil)
	ctx := context.Background()

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	// Act
	override, err := repo.SetOverride(ctx, user.ID, tomorrow, &template.ID)

	// Assert
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}
	if override.ID == 0 {
		t.Error("Expected override ID to be set")
	}
	if override.CalendarDate != tomorrow {
		t.Errorf("Expected date %s, got %s", tomorrow, override.CalendarDate)
	}
	if override.DayTemplateID == nil || *override.DayTemplateID != template.ID {
		t.Errorf("Expected template ID %d, got %v", template.ID, override.DayTemplateID)
	}
}

func TestScheduleRepository_SetOverride_Update(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	template1 := createTestDayTemplate(t, db, user.ID, "Holiday", nil)
	template2 := createTestDayTemplate(t, db, user.ID, "Vacation", nil)
	ctx := context.Background()

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	repo.SetOverride(ctx, user.ID, tomorrow, &template1.ID)

	// Act
	override, err := repo.SetOverride(ctx, user.ID, tomorrow, &template2.ID)

	// Assert
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}
	if override.DayTemplateID == nil || *override.DayTemplateID != template2.ID {
		t.Errorf("Expected template ID %d, got %v", template2.ID, override.DayTemplateID)
	}
}

func TestScheduleRepository_SetOverride_Delete(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	template := createTestDayTemplate(t, db, user.ID, "Holiday", nil)
	ctx := context.Background()

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	repo.SetOverride(ctx, user.ID, tomorrow, &template.ID)

	// Act
	override, err := repo.SetOverride(ctx, user.ID, tomorrow, nil)

	// Assert
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}
	if override.DayTemplateID != nil {
		t.Error("Expected template ID to be nil after deletion")
	}

	// Verify it's actually deleted
	overrides, _ := repo.GetFutureOverrides(ctx, user.ID)
	if len(overrides) != 0 {
		t.Error("Expected override to be deleted")
	}
}

func TestScheduleRepository_SetOverride_DeleteNonExistent(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	// Act
	override, err := repo.SetOverride(ctx, user.ID, tomorrow, nil)

	// Assert
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}
	if override.DayTemplateID != nil {
		t.Error("Expected template ID to be nil")
	}
}

func TestScheduleRepository_MultipleUsers(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewScheduleRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	// Test that each user has their own weekly schedule
	entries := []WeeklyScheduleEntry{
		{DayOfWeek: 0}, {DayOfWeek: 1}, {DayOfWeek: 2},
		{DayOfWeek: 3}, {DayOfWeek: 4}, {DayOfWeek: 5}, {DayOfWeek: 6},
	}

	repo.ReplaceWeeklySchedule(ctx, user1.ID, entries)
	repo.ReplaceWeeklySchedule(ctx, user2.ID, entries)

	// Act
	schedule1, _ := repo.GetWeeklySchedule(ctx, user1.ID)
	schedule2, _ := repo.GetWeeklySchedule(ctx, user2.ID)

	// Assert
	if len(schedule1) != 7 || len(schedule2) != 7 {
		t.Error("Each user should have their own 7-day schedule")
	}
	// Verify they have different IDs (different rows)
	if schedule1[0].ID == schedule2[0].ID {
		t.Error("Users should have separate schedule rows")
	}
}
