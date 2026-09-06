package main

import (
	"context"
	"testing"
)

func TestTemplateSnapshotCreation_OnCreate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	category := createTestCategory(t, db, user.ID, "Work", "#FF5733")

	ctx := context.Background()
	input := DayTemplateInput{
		Name: "Morning Routine",
		SnapshotBlocks: []SnapshotBlockInput{
			{CategoryID: category.ID, StartTime: "08:00:00", DurationMinutes: 60},
			{CategoryID: category.ID, StartTime: "09:00:00", DurationMinutes: 120},
		},
	}

	// Act
	template, err := repo.Create(ctx, input, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if template.ID == 0 {
		t.Fatal("Expected template to have ID")
	}

	// Verify snapshot was created
	var snapshotID int
	err = db.QueryRow(ctx, `
		SELECT id FROM template_snapshots
		WHERE day_template_id = $1 AND user_id = $2
	`, template.ID, user.ID).Scan(&snapshotID)
	if err != nil {
		t.Fatalf("No snapshot found for template: %v", err)
	}

	// Verify snapshot blocks were created
	var blockCount int
	err = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM snapshot_blocks WHERE snapshot_id = $1
	`, snapshotID).Scan(&blockCount)
	if err != nil {
		t.Fatalf("Failed to count snapshot blocks: %v", err)
	}
	if blockCount != 2 {
		t.Errorf("Expected 2 snapshot blocks, got %d", blockCount)
	}
}

func TestTemplateSnapshotCreation_OnUpdate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	category := createTestCategory(t, db, user.ID, "Work", "#FF5733")

	ctx := context.Background()
	createInput := DayTemplateInput{
		Name: "Morning Routine",
		SnapshotBlocks: []SnapshotBlockInput{
			{CategoryID: category.ID, StartTime: "08:00:00", DurationMinutes: 60},
		},
	}
	template, _ := repo.Create(ctx, createInput, user.ID)

	// Count initial snapshots
	var snapshotCountBefore int
	db.QueryRow(ctx, `
		SELECT COUNT(*) FROM template_snapshots WHERE day_template_id = $1
	`, template.ID).Scan(&snapshotCountBefore)

	// Act - update the template
	updateInput := DayTemplateInput{
		Name: "Morning Routine Updated",
		SnapshotBlocks: []SnapshotBlockInput{
			{CategoryID: category.ID, StartTime: "08:00:00", DurationMinutes: 60},
			{CategoryID: category.ID, StartTime: "09:00:00", DurationMinutes: 120},
		},
	}
	updated, err := repo.Update(ctx, template.ID, updateInput, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Morning Routine Updated" {
		t.Errorf("Expected name 'Morning Routine Updated', got '%s'", updated.Name)
	}

	// Verify a new snapshot was created
	var snapshotCountAfter int
	db.QueryRow(ctx, `
		SELECT COUNT(*) FROM template_snapshots WHERE day_template_id = $1
	`, template.ID).Scan(&snapshotCountAfter)

	if snapshotCountAfter != snapshotCountBefore+1 {
		t.Errorf("Expected %d snapshots, got %d", snapshotCountBefore+1, snapshotCountAfter)
	}

	// Verify the latest snapshot has the updated blocks
	var latestSnapshotID int
	err = db.QueryRow(ctx, `
		SELECT id FROM template_snapshots
		WHERE day_template_id = $1
		ORDER BY snapshotted_at DESC
		LIMIT 1
	`, template.ID).Scan(&latestSnapshotID)
	if err != nil {
		t.Fatalf("Failed to get latest snapshot: %v", err)
	}

	var blockCount int
	db.QueryRow(ctx, `
		SELECT COUNT(*) FROM snapshot_blocks WHERE snapshot_id = $1
	`, latestSnapshotID).Scan(&blockCount)
	if blockCount != 2 {
		t.Errorf("Expected 2 blocks in latest snapshot, got %d", blockCount)
	}
}

func TestDayRecordCreation_WithTemplateSnapshot(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	templateRepo := NewDayTemplateRepository(db)
	dayRecordRepo := NewDayRecordRepository(db)
	scheduleRepo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	category := createTestCategory(t, db, user.ID, "Work", "#FF5733")

	ctx := context.Background()

	// Create a template
	templateInput := DayTemplateInput{
		Name: "Weekday Schedule",
		SnapshotBlocks: []SnapshotBlockInput{
			{CategoryID: category.ID, StartTime: "09:00:00", DurationMinutes: 120},
			{CategoryID: category.ID, StartTime: "11:00:00", DurationMinutes: 60},
		},
	}
	template, _ := templateRepo.Create(ctx, templateInput, user.ID)

	// Assign template to Monday (day_of_week = 0)
	weeklySchedule := []WeeklyScheduleEntry{
		{DayOfWeek: 0, DayTemplateID: &template.ID},
		{DayOfWeek: 1, DayTemplateID: nil},
		{DayOfWeek: 2, DayTemplateID: nil},
		{DayOfWeek: 3, DayTemplateID: nil},
		{DayOfWeek: 4, DayTemplateID: nil},
		{DayOfWeek: 5, DayTemplateID: nil},
		{DayOfWeek: 6, DayTemplateID: nil},
	}
	scheduleRepo.ReplaceWeeklySchedule(ctx, user.ID, weeklySchedule)

	// Act - create a day record for a Monday (2026-07-07 is a Tuesday, so use 2026-07-06 Monday)
	dayRecord, err := dayRecordRepo.Create(ctx, user.ID, "2026-07-06")

	// Assert
	if err != nil {
		t.Fatalf("Create day record failed: %v", err)
	}
	if dayRecord.SnapshotID == nil {
		t.Fatal("Expected day record to have a snapshot_id")
	}
	if len(dayRecord.SnapshotBlocks) != 2 {
		t.Errorf("Expected 2 snapshot blocks, got %d", len(dayRecord.SnapshotBlocks))
	}
}
