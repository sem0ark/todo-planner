package main

import (
	"context"
	"testing"
	"time"
)

func TestDayRecordRepository_Create(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	scheduleRepo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	template := createTestDayTemplate(t, db, user.ID, "Workday", nil)
	ctx := context.Background()

	// Set up weekly schedule
	entries := make([]WeeklyScheduleEntry, 7)
	for i := 0; i < 7; i++ {
		entries[i] = WeeklyScheduleEntry{DayOfWeek: i, DayTemplateID: &template.ID}
	}
	_, err := scheduleRepo.ReplaceWeeklySchedule(ctx, user.ID, entries)
	if err != nil {
		t.Fatalf("Failed to set up schedule: %v", err)
	}

	calendarDate := "2026-07-07"

	// Act
	record, err := repo.Create(ctx, user.ID, calendarDate)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if record.ID == 0 {
		t.Error("Expected non-zero ID")
	}
	if record.CalendarDate != calendarDate {
		t.Errorf("Expected calendar_date '%s', got '%s'", calendarDate, record.CalendarDate)
	}
	// Note: snapshot_id may be nil if no snapshot was created for the template
	// This is expected behavior when a template has no planned blocks
	if len(record.ActualBlocks) != 0 {
		t.Errorf("Expected 0 actual blocks, got %d", len(record.ActualBlocks))
	}
}

func TestDayRecordRepository_Create_NoTemplate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	calendarDate := "2026-07-07"

	// Act
	record, err := repo.Create(ctx, user.ID, calendarDate)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if record.SnapshotID != nil {
		t.Error("Expected snapshot_id to be nil when no template assigned")
	}
	if len(record.SnapshotBlocks) != 0 {
		t.Errorf("Expected 0 snapshot blocks, got %d", len(record.SnapshotBlocks))
	}
}

func TestDayRecordRepository_Create_DuplicateDate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	calendarDate := "2026-07-07"
	_, err := repo.Create(ctx, user.ID, calendarDate)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Act
	_, err = repo.Create(ctx, user.ID, calendarDate)

	// Assert
	if err == nil {
		t.Error("Expected error for duplicate calendar_date")
	}
}

func TestDayRecordRepository_FindByID(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	calendarDate := "2026-07-07"
	created, err := repo.Create(ctx, user.ID, calendarDate)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Act
	found, err := repo.FindByID(ctx, created.ID, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, found.ID)
	}
	if found.CalendarDate != calendarDate {
		t.Errorf("Expected calendar_date '%s', got '%s'", calendarDate, found.CalendarDate)
	}
}

func TestDayRecordRepository_FindByID_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	_, err := repo.FindByID(ctx, 9999, user.ID)

	// Assert
	if err == nil {
		t.Error("Expected error for non-existent record")
	}
}

func TestDayRecordRepository_FindByDateRange(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Create multiple records
	dates := []string{"2026-07-01", "2026-07-05", "2026-07-10", "2026-07-15"}
	for _, date := range dates {
		_, err := repo.Create(ctx, user.ID, date)
		if err != nil {
			t.Fatalf("Create failed for date %s: %v", date, err)
		}
	}

	// Act
	records, err := repo.FindByDateRange(ctx, user.ID, "2026-07-05", "2026-07-10")

	// Assert
	if err != nil {
		t.Fatalf("FindByDateRange failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}
	if records[0].CalendarDate != "2026-07-05" {
		t.Errorf("Expected first record date '2026-07-05', got '%s'", records[0].CalendarDate)
	}
	if records[1].CalendarDate != "2026-07-10" {
		t.Errorf("Expected second record date '2026-07-10', got '%s'", records[1].CalendarDate)
	}
}

func TestDayRecordRepository_FindByDateRange_Empty(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	records, err := repo.FindByDateRange(ctx, user.ID, "2026-07-01", "2026-07-31")

	// Assert
	if err != nil {
		t.Fatalf("FindByDateRange failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("Expected 0 records, got %d", len(records))
	}
}

func TestDayRecordRepository_ResolveTemplateForDate_WithOverride(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	scheduleRepo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	weekdayTemplate := createTestDayTemplate(t, db, user.ID, "Workday", nil)
	overrideTemplate := createTestDayTemplate(t, db, user.ID, "Special", nil)
	ctx := context.Background()

	// Set up weekly schedule
	entries := make([]WeeklyScheduleEntry, 7)
	for i := 0; i < 7; i++ {
		entries[i] = WeeklyScheduleEntry{DayOfWeek: i, DayTemplateID: &weekdayTemplate.ID}
	}
	_, err := scheduleRepo.ReplaceWeeklySchedule(ctx, user.ID, entries)
	if err != nil {
		t.Fatalf("Failed to set up schedule: %v", err)
	}

	// Set up override
	date := "2026-07-07"
	override, err := scheduleRepo.SetOverride(ctx, user.ID, date, &overrideTemplate.ID)
	if err != nil {
		t.Fatalf("Failed to set up override: %v", err)
	}
	t.Logf("Created override: %+v", override)

	// Verify override was created
	var storedTemplateID *int
	err = db.QueryRow(ctx, "SELECT day_template_id FROM schedule_overrides WHERE user_id=$1 AND calendar_date=$2", user.ID, date).Scan(&storedTemplateID)
	if err != nil {
		t.Fatalf("Failed to query override: %v", err)
	}
	t.Logf("Stored override template_id: %v", storedTemplateID)

	// Act
	templateID, err := repo.resolveTemplateForDate(ctx, user.ID, date)

	// Assert
	if err != nil {
		t.Fatalf("resolveTemplateForDate failed: %v", err)
	}
	if templateID == nil {
		t.Fatal("Expected non-nil template ID")
	}
	t.Logf("weekdayTemplate.ID=%d, overrideTemplate.ID=%d, resolved=%d", weekdayTemplate.ID, overrideTemplate.ID, *templateID)
	if *templateID != overrideTemplate.ID {
		t.Errorf("Expected template ID %d (override), got %d", overrideTemplate.ID, *templateID)
	}
}

func TestDayRecordRepository_ResolveTemplateForDate_WithoutOverride(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	scheduleRepo := NewScheduleRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	template := createTestDayTemplate(t, db, user.ID, "Workday", nil)
	ctx := context.Background()

	// Set up weekly schedule
	entries := make([]WeeklyScheduleEntry, 7)
	for i := 0; i < 7; i++ {
		entries[i] = WeeklyScheduleEntry{DayOfWeek: i, DayTemplateID: &template.ID}
	}
	_, err := scheduleRepo.ReplaceWeeklySchedule(ctx, user.ID, entries)
	if err != nil {
		t.Fatalf("Failed to set up schedule: %v", err)
	}

	// Act
	date := "2026-07-07" // Monday
	templateID, err := repo.resolveTemplateForDate(ctx, user.ID, date)

	// Assert
	if err != nil {
		t.Fatalf("resolveTemplateForDate failed: %v", err)
	}
	if templateID == nil {
		t.Fatal("Expected non-nil template ID")
	}
	if *templateID != template.ID {
		t.Errorf("Expected template ID %d, got %d", template.ID, *templateID)
	}
}

func TestDayRecordRepository_MultipleUsers(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	calendarDate := "2026-07-07"
	record1, err := repo.Create(ctx, user1.ID, calendarDate)
	if err != nil {
		t.Fatalf("Create for user1 failed: %v", err)
	}
	record2, err := repo.Create(ctx, user2.ID, calendarDate)
	if err != nil {
		t.Fatalf("Create for user2 failed: %v", err)
	}

	// Act - user1 should only see their record
	found, err := repo.FindByID(ctx, record1.ID, user1.ID)
	if err != nil {
		t.Fatalf("FindByID for user1 failed: %v", err)
	}
	if found.ID != record1.ID {
		t.Errorf("Expected user1 to find record %d, got %d", record1.ID, found.ID)
	}

	// Act - user1 should not find user2's record
	_, err = repo.FindByID(ctx, record2.ID, user1.ID)

	// Assert
	if err == nil {
		t.Error("Expected error when user1 tries to access user2's record")
	}
}

func TestComputeActualBlocks_Empty(t *testing.T) {
	// Arrange
	events := []DayEvent{}
	referenceTime := parseTime("2026-07-20T17:00:00Z")

	// Act
	blocks := computeActualBlocks(events, referenceTime)

	// Assert
	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(blocks))
	}
}

func TestComputeActualBlocks_OnlyConfirmations(t *testing.T) {
	// Arrange
	events := []DayEvent{
		{
			ID:         1,
			EventType:  "confirmation",
			OccurredAt: parseTime("2026-07-20T09:00:00Z"),
		},
		{
			ID:         2,
			EventType:  "confirmation",
			OccurredAt: parseTime("2026-07-20T10:00:00Z"),
		},
	}
	referenceTime := parseTime("2026-07-20T17:00:00Z")

	// Act
	blocks := computeActualBlocks(events, referenceTime)

	// Assert - confirmations should be skipped
	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks for confirmation-only events, got %d", len(blocks))
	}
}

func TestComputeActualBlocks_SingleTransition(t *testing.T) {
	// Arrange
	categoryID := 5
	startTime := parseTime("2026-07-20T09:00:00Z")
	referenceTime := parseTime("2026-07-20T17:00:00Z")

	events := []DayEvent{
		{
			ID:         1,
			EventType:  "transition",
			OccurredAt: startTime,
			CategoryID: &categoryID,
		},
	}

	// Act
	blocks := computeActualBlocks(events, referenceTime)

	// Assert
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
	}
	if blocks[0].CategoryID == nil || *blocks[0].CategoryID != categoryID {
		t.Errorf("Expected category_id %d, got %v", categoryID, blocks[0].CategoryID)
	}
	if blocks[0].DurationMinutes != 480 { // 8 hours = 480 minutes
		t.Errorf("Expected 480 minutes, got %d", blocks[0].DurationMinutes)
	}
}

func TestComputeActualBlocks_MultipleTransitions(t *testing.T) {
	// Arrange
	category1 := 5
	category2 := 10
	referenceTime := parseTime("2026-07-20T17:00:00Z")

	events := []DayEvent{
		{
			ID:         1,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T09:00:00Z"),
			CategoryID: &category1,
		},
		{
			ID:         2,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T12:00:00Z"),
			CategoryID: &category2,
		},
		{
			ID:         3,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T13:00:00Z"),
			CategoryID: &category1,
		},
	}

	// Act
	blocks := computeActualBlocks(events, referenceTime)

	// Assert
	if len(blocks) != 3 {
		t.Fatalf("Expected 3 blocks, got %d", len(blocks))
	}

	// First block: 9:00 to 12:00 (180 minutes)
	if *blocks[0].CategoryID != category1 {
		t.Errorf("Block 0: Expected category %d, got %d", category1, *blocks[0].CategoryID)
	}
	if blocks[0].DurationMinutes != 180 {
		t.Errorf("Block 0: Expected 180 minutes, got %d", blocks[0].DurationMinutes)
	}

	// Second block: 12:00 to 13:00 (60 minutes)
	if *blocks[1].CategoryID != category2 {
		t.Errorf("Block 1: Expected category %d, got %d", category2, *blocks[1].CategoryID)
	}
	if blocks[1].DurationMinutes != 60 {
		t.Errorf("Block 1: Expected 60 minutes, got %d", blocks[1].DurationMinutes)
	}

	// Third block: 13:00 to 17:00 (240 minutes)
	if *blocks[2].CategoryID != category1 {
		t.Errorf("Block 2: Expected category %d, got %d", category1, *blocks[2].CategoryID)
	}
	if blocks[2].DurationMinutes != 240 {
		t.Errorf("Block 2: Expected 240 minutes, got %d", blocks[2].DurationMinutes)
	}
}

func TestComputeActualBlocks_MixedEvents(t *testing.T) {
	// Arrange - transitions interspersed with confirmations
	category1 := 5
	category2 := 10
	referenceTime := parseTime("2026-07-20T17:00:00Z")

	events := []DayEvent{
		{
			ID:         1,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T09:00:00Z"),
			CategoryID: &category1,
		},
		{
			ID:         2,
			EventType:  "confirmation",
			OccurredAt: parseTime("2026-07-20T10:30:00Z"),
		},
		{
			ID:         3,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T12:00:00Z"),
			CategoryID: &category2,
		},
		{
			ID:         4,
			EventType:  "confirmation",
			OccurredAt: parseTime("2026-07-20T14:00:00Z"),
		},
	}

	// Act
	blocks := computeActualBlocks(events, referenceTime)

	// Assert
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks (confirmations skipped), got %d", len(blocks))
	}

	// First block: 9:00 to 12:00 (180 minutes)
	if blocks[0].DurationMinutes != 180 {
		t.Errorf("Block 0: Expected 180 minutes, got %d", blocks[0].DurationMinutes)
	}

	// Second block: 12:00 to 17:00 (300 minutes)
	if blocks[1].DurationMinutes != 300 {
		t.Errorf("Block 1: Expected 300 minutes, got %d", blocks[1].DurationMinutes)
	}
}

func TestComputeActualBlocks_ZeroDurationBlocks(t *testing.T) {
	// Arrange - transitions at same time should be skipped
	category1 := 5
	referenceTime := parseTime("2026-07-20T17:00:00Z")

	events := []DayEvent{
		{
			ID:         1,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T09:00:00Z"),
			CategoryID: &category1,
		},
		{
			ID:         2,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T09:00:00Z"),
			CategoryID: &category1,
		},
	}

	// Act
	blocks := computeActualBlocks(events, referenceTime)

	// Assert - zero-duration completed blocks are skipped, but the current block remains open.
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
	}
	if blocks[0].DurationMinutes != 480 {
		t.Errorf("Expected 480 minutes, got %d", blocks[0].DurationMinutes)
	}
}

func TestComputeActualBlocks_CategoryID(t *testing.T) {
	referenceTime := parseTime("2026-07-20T17:00:00Z")
	category1 := 1
	category2 := 2

	events := []DayEvent{
		{
			ID:         1,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T09:00:00Z"),
			CategoryID: &category1,
		},
		{
			ID:         2,
			EventType:  "transition",
			OccurredAt: parseTime("2026-07-20T12:00:00Z"),
			CategoryID: &category2,
		},
	}

	// Act
	blocks := computeActualBlocks(events, referenceTime)

	// Assert
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].CategoryID == nil {
		t.Errorf("Expected category_id, got nil")
	}
}

func TestComputeActualBlocks_ExcludesSubMinuteOngoingBlock(t *testing.T) {
	// Arrange
	categoryID := 1
	startTime := parseTime("2026-07-20T09:00:00Z")
	events := []DayEvent{{
		EventType:  "transition",
		CategoryID: &categoryID,
		OccurredAt: startTime,
	}}

	// Act
	blocks := computeActualBlocks(events, startTime.Add(59*time.Second))

	// Assert
	if len(blocks) != 1 {
		t.Fatalf("Expected the open block before one full minute, got %d", len(blocks))
	}
	if !blocks[0].IsOpen || blocks[0].DurationMinutes != 0 {
		t.Fatalf("Expected a zero-duration open block, got %+v", blocks[0])
	}
}

func TestDayRecordRepository_CreateEvents_RollsBackPartialBatch(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := NewCategoryRepository(db).Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	record, _ := repo.Create(context.Background(), user.ID, "2026-07-07")
	invalidCategoryID := 99999
	inputs := []DayEventInput{
		{EventType: "transition", CategoryID: &category.ID, OccurredAt: parseTime("2026-07-07T09:00:00Z")},
		{EventType: "transition", CategoryID: &invalidCategoryID, OccurredAt: parseTime("2026-07-07T10:00:00Z")},
	}

	// Act
	_, _, err := repo.CreateEvents(context.Background(), record.ID, user.ID, inputs)

	// Assert
	if err == nil {
		t.Fatal("Expected CreateEvents to fail")
	}
	var eventCount int
	if err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM day_events WHERE day_record_id = $1", record.ID).Scan(&eventCount); err != nil {
		t.Fatalf("Failed to count day events: %v", err)
	}
	if eventCount != 0 {
		t.Errorf("Expected no events after rollback, got %d", eventCount)
	}
}

func TestDayRecordRepository_ReplaceActualBlocks_RollsBackPartialBatch(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayRecordRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := NewCategoryRepository(db).Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	record, _ := repo.Create(context.Background(), user.ID, "2026-07-07")
	_, err := repo.ReplaceActualBlocks(context.Background(), record.ID, user.ID, []ActualBlockInput{
		{CategoryID: &category.ID, BlockType: "actual", StartTime: "09:00:00", DurationMinutes: 60},
	})
	if err != nil {
		t.Fatalf("Failed to create initial actual block: %v", err)
	}
	invalidCategoryID := 99999

	// Act
	_, err = repo.ReplaceActualBlocks(context.Background(), record.ID, user.ID, []ActualBlockInput{
		{CategoryID: &category.ID, BlockType: "actual", StartTime: "10:00:00", DurationMinutes: 60},
		{CategoryID: &invalidCategoryID, BlockType: "actual", StartTime: "11:00:00", DurationMinutes: 60},
	})

	// Assert
	if err == nil {
		t.Fatal("Expected ReplaceActualBlocks to fail")
	}
	var blockCount int
	if err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM actual_blocks WHERE day_record_id = $1", record.ID).Scan(&blockCount); err != nil {
		t.Fatalf("Failed to count actual blocks: %v", err)
	}
	if blockCount != 1 {
		t.Errorf("Expected original block to remain after rollback, got %d blocks", blockCount)
	}
	var startTime string
	if err := db.QueryRow(context.Background(), "SELECT start_time::text FROM actual_blocks WHERE day_record_id = $1", record.ID).Scan(&startTime); err != nil {
		t.Fatalf("Failed to load original actual block: %v", err)
	}
	if startTime != "09:00:00" {
		t.Errorf("Expected original block at 09:00:00, got %s", startTime)
	}
}

// Helper function to parse time strings for tests
func parseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic(err)
	}
	return t
}
