package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func computeActualTimelineForTest(t *testing.T, events []DayEvent, referenceTime time.Time) []ComputedBlock {
	t.Helper()
	blocks, err := computeTimeline(events)
	if err != nil {
		t.Fatalf("computeTimeline failed: %v", err)
	}
	actualBlocks := make([]ComputedBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.BlockType == "actual" {
			actualBlocks = append(actualBlocks, block)
		}
	}
	return actualBlocks
}

func TestDayTrackingService_CreateDayRecord(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
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

	calendarDate := mustCalendarDate("2026-07-07")

	// Act
	record, err := newDayTrackingServiceForTest(db).CreateDayRecord(ctx, user.ID, calendarDate)

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

func TestDayTrackingService_UpdateTemplateByDate(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "update-day-template", "password123")
	category := createTestCategory(t, database, user.ID, "Work", "#4A90D9")
	templateRepository := NewDayTemplateRepository(database)
	scheduledTemplate, err := templateRepository.Create(context.Background(), DayTemplateInput{
		Name: "Scheduled",
		SnapshotBlocks: []SnapshotBlockInput{{
			CategoryID:      category.ID,
			StartTime:       mustScheduleTime("09:00:00"),
			DurationMinutes: 120,
		}},
	}, user.ID)
	if err != nil {
		t.Fatalf("failed to create scheduled template: %v", err)
	}
	requestedTemplate, err := templateRepository.Create(context.Background(), DayTemplateInput{
		Name: "Requested",
		SnapshotBlocks: []SnapshotBlockInput{{
			CategoryID:      category.ID,
			StartTime:       mustScheduleTime("13:00:00"),
			DurationMinutes: 60,
		}},
	}, user.ID)
	if err != nil {
		t.Fatalf("failed to create requested template: %v", err)
	}
	calendarDate := calendarDateRelativeToToday(t, database, 1)
	weeklySchedule := make([]WeeklyScheduleEntry, 7)
	for weekday := range weeklySchedule {
		weeklySchedule[weekday] = WeeklyScheduleEntry{
			DayOfWeek:     weekday,
			DayTemplateID: &scheduledTemplate.ID,
		}
	}
	if _, err := NewScheduleRepository(database).ReplaceWeeklySchedule(
		context.Background(), user.ID, weeklySchedule,
	); err != nil {
		t.Fatalf("failed to set up weekly schedule: %v", err)
	}
	if _, err := newDayTrackingServiceForTest(database).CreateDayRecord(
		context.Background(), user.ID, calendarDate,
	); err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	service := newDayTrackingServiceForTest(database)

	requestedRecord, err := service.UpdateTemplateByDate(
		context.Background(), user.ID, calendarDate, &requestedTemplate.ID,
	)
	if err != nil {
		t.Fatalf("failed to update day record with requested template: %v", err)
	}
	if requestedRecord.DayTemplateID == nil || *requestedRecord.DayTemplateID != requestedTemplate.ID {
		t.Fatalf("expected requested template ID %d, got %v", requestedTemplate.ID, requestedRecord.DayTemplateID)
	}
	if requestedRecord.SnapshotID == nil || *requestedRecord.SnapshotID != requestedTemplate.CurrentSnapshot.ID {
		t.Fatalf("expected requested snapshot ID %d, got %v", requestedTemplate.CurrentSnapshot.ID, requestedRecord.SnapshotID)
	}
	if len(requestedRecord.SnapshotBlocks) != 1 || requestedRecord.SnapshotBlocks[0].DurationMinutes != 60 {
		t.Fatalf("expected requested template snapshot blocks to be hydrated, got %+v", requestedRecord.SnapshotBlocks)
	}

	scheduledRecord, err := service.UpdateTemplateByDate(
		context.Background(), user.ID, calendarDate, nil,
	)
	if err != nil {
		t.Fatalf("failed to resolve scheduled template: %v", err)
	}
	if scheduledRecord.DayTemplateID == nil || *scheduledRecord.DayTemplateID != scheduledTemplate.ID {
		t.Fatalf("expected scheduled template ID %d, got %v", scheduledTemplate.ID, scheduledRecord.DayTemplateID)
	}
	if scheduledRecord.SnapshotID == nil || *scheduledRecord.SnapshotID != scheduledTemplate.CurrentSnapshot.ID {
		t.Fatalf("expected scheduled snapshot ID %d, got %v", scheduledTemplate.CurrentSnapshot.ID, scheduledRecord.SnapshotID)
	}
}

func TestDayTrackingService_UpdateTemplateByDateRejectsInvalidTemplate(t *testing.T) {
	database := setupTestDB(t)
	owner := createTestUser(t, database, "template-owner", "password123")
	otherUser := createTestUser(t, database, "template-other-user", "password123")
	otherUsersTemplate := createTestDayTemplate(t, database, otherUser.ID, "Private", nil)
	calendarDate := calendarDateRelativeToToday(t, database, 1)
	createdRecord, err := newDayTrackingServiceForTest(database).CreateDayRecord(
		context.Background(), owner.ID, calendarDate,
	)
	if err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	missingTemplateID := otherUsersTemplate.ID + 10000

	for _, requestedTemplateID := range []int{otherUsersTemplate.ID, missingTemplateID} {
		_, updateError := newDayTrackingServiceForTest(database).UpdateTemplateByDate(
			context.Background(), owner.ID, calendarDate, &requestedTemplateID,
		)
		if !errors.Is(updateError, ErrDayTemplateNotFound) {
			t.Errorf("expected ErrDayTemplateNotFound for template %d, got %v", requestedTemplateID, updateError)
		}
	}

	unchangedRecord, err := NewDayRecordRepository(database).FindByID(
		context.Background(), createdRecord.ID, owner.ID,
	)
	if err != nil {
		t.Fatalf("failed to reload unchanged day record: %v", err)
	}
	if unchangedRecord.DayTemplateID != nil || unchangedRecord.SnapshotID != nil {
		t.Fatalf("invalid template update changed the record: %+v", unchangedRecord)
	}
}

func TestDayTrackingService_UpdateTemplateByDateReturnsMissingOrPastErrors(t *testing.T) {
	testCases := []struct {
		name          string
		dayOffset     int
		createRecord  bool
		expectedError error
	}{
		{
			name:          "missing record",
			dayOffset:     1,
			expectedError: ErrDayRecordNotFound,
		},
		{
			name:          "past record",
			dayOffset:     -1,
			createRecord:  true,
			expectedError: ErrDayRecordPast,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database := setupTestDB(t)
			user := createTestUser(t, database, "update-template-"+testCase.name, "password123")
			calendarDate := calendarDateRelativeToToday(t, database, testCase.dayOffset)
			if testCase.createRecord {
				createDayRecordForRepositoryTest(
					t,
					database,
					NewDayRecordRepository(database),
					user.ID,
					calendarDate,
				)
			}

			_, updateError := newDayTrackingServiceForTest(database).UpdateTemplateByDate(
				context.Background(), user.ID, calendarDate, nil,
			)
			if !errors.Is(updateError, testCase.expectedError) {
				t.Fatalf("expected error %v, got %v", testCase.expectedError, updateError)
			}
		})
	}
}

func TestDayTrackingService_CreateDayRecordWithoutTemplate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	calendarDate := mustCalendarDate("2026-07-07")

	// Act
	record, err := newDayTrackingServiceForTest(db).CreateDayRecord(ctx, user.ID, calendarDate)

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

func TestDayTrackingService_CreateEventsByDate_ConcurrentDateCreation(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "concurrent-event-user", "password123")
	device, err := NewDeviceRepository(database).Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	for range 2 {
		go func() {
			defer waitGroup.Done()
			<-start
			_, createError := newDayTrackingServiceForTest(database).CreateEventsByDate(
				context.Background(),
				user.ID,
				mustCalendarDate("2026-09-06"),
				device.ID,
				nil,
			)
			results <- createError
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	for createError := range results {
		if createError != nil {
			t.Fatalf("concurrent event creation failed: %v", createError)
		}
	}

	var recordCount int
	if err := database.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM day_records WHERE user_id = $1 AND calendar_date = $2
	`, user.ID, mustCalendarDate("2026-09-06")).Scan(&recordCount); err != nil {
		t.Fatalf("failed to count created records: %v", err)
	}
	if recordCount != 1 {
		t.Fatalf("expected one day record, got %d", recordCount)
	}
}

func TestDayTrackingService_CreateDayRecordStartsWithUnlockedOffset(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "unlocked-offset-user", "password123")

	record, err := newDayTrackingServiceForTest(database).CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-09-06"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if record.TimezoneOffsetMinutes != nil {
		t.Fatalf("expected nil timezone offset, got %v", *record.TimezoneOffsetMinutes)
	}
	if record.TimezoneOffsetLocked {
		t.Fatal("expected newly created day to have an unlocked timezone offset")
	}
}

func TestDayTrackingService_LocksFirstEligibleEventOffset(t *testing.T) {
	database := setupTestDB(t)
	repository := NewDayRecordRepository(database)
	user := createTestUser(t, database, "first-offset-user", "password123")
	category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
	device, err := NewDeviceRepository(database).Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	events := []DayEventInput{
		{
			ClientEventID:   "transition-first",
			EventType:       "transition",
			CategoryID:      &category.ID,
			OccurredAt:      parseTime("2026-09-06T08:00:00Z"),
			OccurredAtLocal: "2026-09-06T10:00:00+02:00",
		},
		{
			ClientEventID:   "transition-second",
			EventType:       "transition",
			CategoryID:      &category.ID,
			OccurredAt:      parseTime("2026-09-06T09:00:00Z"),
			OccurredAtLocal: "2026-09-06T04:00:00-05:00",
		},
	}
	_, err = newDayTrackingServiceForTest(database).CreateEventsByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06"), device.ID, events)
	if err != nil {
		t.Fatalf("CreateEventsByDate failed: %v", err)
	}

	record, err := repository.FindByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06"))
	if err != nil {
		t.Fatalf("FindByDate failed: %v", err)
	}
	if record.TimezoneOffsetMinutes == nil || *record.TimezoneOffsetMinutes != 120 {
		t.Fatalf("expected first event offset 120, got %v", record.TimezoneOffsetMinutes)
	}
	if !record.TimezoneOffsetLocked {
		t.Fatal("expected timezone offset to be locked")
	}
}

func TestDayTrackingService_AmendmentOffsetValidation(t *testing.T) {
	testCases := []struct {
		name        string
		correctedAt string
		expectedErr error
	}{
		{name: "matching offset", correctedAt: "2026-09-06T11:00:00+02:00"},
		{name: "mismatched offset", correctedAt: "2026-09-06T04:00:00-05:00", expectedErr: ErrAmendmentOffsetMismatch},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database := setupTestDB(t)
			user := createTestUser(t, database, "amendment-offset-"+testCase.name, "password123")
			category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
			correctedAt := parseTime("2026-09-06T09:00:00Z")
			device, err := NewDeviceRepository(database).Create(context.Background(), user.ID, "desktop")
			if err != nil {
				t.Fatalf("failed to create device: %v", err)
			}
			_, err = newDayTrackingServiceForTest(database).CreateEventsByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06"), device.ID, []DayEventInput{{
				ClientEventID:   "target-transition",
				EventType:       "transition",
				CategoryID:      &category.ID,
				OccurredAt:      parseTime("2026-09-06T08:00:00Z"),
				OccurredAtLocal: "2026-09-06T10:00:00+02:00",
			}})
			if err != nil {
				t.Fatalf("failed to create target event: %v", err)
			}

			_, err = newDayTrackingServiceForTest(database).CreateEventsByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06"), device.ID, []DayEventInput{{
				ClientEventID:       "amendment-event",
				EventType:           "amendment",
				CategoryID:          &category.ID,
				OccurredAt:          parseTime("2026-09-06T12:00:00Z"),
				OccurredAtLocal:     "2026-09-06T14:00:00+02:00",
				TargetClientEventID: "target-transition",
				CorrectedAt:         &correctedAt,
				CorrectedAtLocal:    &testCase.correctedAt,
			}})
			if err != testCase.expectedErr {
				t.Fatalf("expected error %v, got %v", testCase.expectedErr, err)
			}
		})
	}
}

func TestDayTrackingService_ManualBlocksLockOffsetOnlyOnce(t *testing.T) {
	database := setupTestDB(t)
	repository := NewDayRecordRepository(database)
	user := createTestUser(t, database, "manual-offset-user", "password123")
	record, err := newDayTrackingServiceForTest(database).CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-09-06"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	blocks := []ActualBlockInput{{BlockType: "blank", StartTime: mustScheduleTime("09:00:00"), DurationMinutes: 30}}
	initialOffsetMinutes := 120
	secondOffsetMinutes := -300

	_, err = newDayTrackingServiceForTest(database).ReplaceActualBlocks(context.Background(), record.ID, user.ID, blocks, &initialOffsetMinutes)
	if err != nil {
		t.Fatalf("initial ReplaceActualBlocks failed: %v", err)
	}
	_, err = newDayTrackingServiceForTest(database).ReplaceActualBlocks(context.Background(), record.ID, user.ID, blocks, &secondOffsetMinutes)
	if err != nil {
		t.Fatalf("locked ReplaceActualBlocks failed: %v", err)
	}

	reloaded, err := repository.FindByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06"))
	if err != nil {
		t.Fatalf("FindByDate failed: %v", err)
	}
	if reloaded.TimezoneOffsetMinutes == nil || *reloaded.TimezoneOffsetMinutes != 120 {
		t.Fatalf("expected original offset 120 to remain locked, got %v", reloaded.TimezoneOffsetMinutes)
	}
}

func TestDayTrackingService_CreateDayRecordRejectsDuplicateDate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	calendarDate := mustCalendarDate("2026-07-07")
	_, err := newDayTrackingServiceForTest(db).CreateDayRecord(ctx, user.ID, calendarDate)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Act
	_, err = newDayTrackingServiceForTest(db).CreateDayRecord(ctx, user.ID, calendarDate)

	// Assert
	if err == nil {
		t.Error("Expected error for duplicate calendar_date")
	}
}

func TestDayTrackingService_CreateDayRecordUsesDateOverride(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
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
	date := mustCalendarDate("2026-07-07")
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
	record, err := newDayTrackingServiceForTest(db).CreateDayRecord(ctx, user.ID, date)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if record.DayTemplateID == nil || *record.DayTemplateID != overrideTemplate.ID {
		t.Errorf("Expected template ID %d (override), got %v", overrideTemplate.ID, record.DayTemplateID)
	}
}

func TestDayTrackingService_CreateDayRecordUsesWeeklySchedule(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
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
	date := mustCalendarDate("2026-07-07") // Monday
	record, err := newDayTrackingServiceForTest(db).CreateDayRecord(ctx, user.ID, date)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if record.DayTemplateID == nil || *record.DayTemplateID != template.ID {
		t.Errorf("Expected template ID %d, got %v", template.ID, record.DayTemplateID)
	}
}

func TestComputeActualBlocks_Empty(t *testing.T) {
	// Arrange
	events := []DayEvent{}
	referenceTime := parseTime("2026-07-20T17:00:00Z")

	// Act
	blocks := computeActualTimelineForTest(t, events, referenceTime)

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
	blocks := computeActualTimelineForTest(t, events, referenceTime)

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
	blocks := computeActualTimelineForTest(t, events, referenceTime)

	// Assert
	if len(blocks) != 0 {
		t.Errorf("Expected no materialized blocks, got %d", len(blocks))
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
	blocks := computeActualTimelineForTest(t, events, referenceTime)

	// Assert
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
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
			CategoryID: &category2,
			OccurredAt: parseTime("2026-07-20T14:00:00Z"),
		},
	}

	// Act
	blocks := computeActualTimelineForTest(t, events, referenceTime)

	// Assert
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks (confirmations skipped), got %d", len(blocks))
	}

	// First block: 9:00 to 12:00 (180 minutes)
	if blocks[0].DurationMinutes != 180 {
		t.Errorf("Block 0: Expected 180 minutes, got %d", blocks[0].DurationMinutes)
	}

	// Second block ends at the latest confirmation.
	if blocks[1].DurationMinutes != 120 {
		t.Errorf("Block 1: Expected 120 minutes, got %d", blocks[1].DurationMinutes)
	}
}

func TestComputeActualBlocks_EqualTimestampTransitionsKeepLastObservation(t *testing.T) {
	// Arrange - equal effective transition times are collapsed deterministically.
	category1 := 5

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
	blocks, err := computeTimeline(events)

	// Assert
	if err != nil {
		t.Fatalf("Expected equal timestamps to be accepted, got %v", err)
	}
	if len(blocks) != 0 {
		t.Fatalf("Expected no block without a later observation, got %+v", blocks)
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
	blocks := computeActualTimelineForTest(t, events, referenceTime)

	// Assert
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
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
	blocks := computeActualTimelineForTest(t, events, startTime.Add(59*time.Second))

	// Assert: a transition without a later confirmation produces no actual block.
	if len(blocks) != 0 {
		t.Fatalf("Expected no actual blocks, got %+v", blocks)
	}
}

func TestDayTrackingService_CreateEvents_RollsBackPartialBatch(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := NewCategoryRepository(db).Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	record, _ := newDayTrackingServiceForTest(db).CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-07-07"))
	invalidCategoryID := 99999
	inputs := []DayEventInput{
		{EventType: "transition", CategoryID: &category.ID, OccurredAt: parseTime("2026-07-07T09:00:00Z")},
		{EventType: "transition", CategoryID: &invalidCategoryID, OccurredAt: parseTime("2026-07-07T10:00:00Z")},
	}

	// Act
	_, _, err := newDayTrackingServiceForTest(db).CreateEvents(context.Background(), record.ID, user.ID, inputs)

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

func TestDayTrackingService_CreateEventsByDate_AmendmentOrdering(t *testing.T) {
	testCases := []struct {
		name             string
		targetOccurredAt string
		amendmentAt      string
	}{
		{name: "target before amendment", targetOccurredAt: "09:00:00", amendmentAt: "10:00:00"},
		{name: "amendment before target", targetOccurredAt: "10:00:00", amendmentAt: "09:00:00"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database := setupTestDB(t)
			user := createTestUser(t, database, "amendment-order-"+testCase.name, "password123")
			category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
			device, err := NewDeviceRepository(database).Create(context.Background(), user.ID, "desktop")
			if err != nil {
				t.Fatalf("failed to create device: %v", err)
			}
			targetOccurredAt := parseTime("2026-09-06T" + testCase.targetOccurredAt + "Z")
			amendmentOccurredAt := parseTime("2026-09-06T" + testCase.amendmentAt + "Z")
			correctedAtLocal := "2026-09-06T08:30:00+00:00"
			correctedAt := parseTime("2026-09-06T08:30:00Z")
			inputs := []DayEventInput{
				{
					ClientEventID:   "target-event",
					EventType:       "transition",
					CategoryID:      &category.ID,
					OccurredAt:      targetOccurredAt,
					OccurredAtLocal: "2026-09-06T10:00:00+00:00",
				},
				{
					ClientEventID:       "amendment-event",
					EventType:           "amendment",
					OccurredAt:          amendmentOccurredAt,
					OccurredAtLocal:     "2026-09-06T11:00:00+00:00",
					TargetClientEventID: "target-event",
					CorrectedAt:         &correctedAt,
					CorrectedAtLocal:    &correctedAtLocal,
				},
			}
			if amendmentOccurredAt.Before(targetOccurredAt) {
				inputs[0], inputs[1] = inputs[1], inputs[0]
			}

			result, err := newDayTrackingServiceForTest(database).CreateEventsByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06"), device.ID, inputs)
			if err != nil {
				t.Fatalf("CreateEventsByDate failed: %v", err)
			}
			if len(result.AcceptedEvents) != 2 {
				t.Fatalf("expected both events to be accepted, got %+v", result.AcceptedEvents)
			}
		})
	}
}

func TestDayTrackingService_CreateEventsByDate_ConcurrentDuplicate(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "concurrent-duplicate-user", "password123")
	category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
	device, err := NewDeviceRepository(database).Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	input := []DayEventInput{{
		ClientEventID:   "concurrent-event",
		EventType:       "transition",
		CategoryID:      &category.ID,
		OccurredAt:      parseTime("2026-09-06T09:00:00Z"),
		OccurredAtLocal: "2026-09-06T09:00:00+00:00",
	}}

	start := make(chan struct{})
	results := make(chan *DateEventResult, 2)
	errors := make(chan error, 2)
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	for range 2 {
		go func() {
			defer waitGroup.Done()
			<-start
			result, createError := newDayTrackingServiceForTest(database).CreateEventsByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06"), device.ID, input)
			results <- result
			errors <- createError
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)
	close(errors)

	acceptedCount := 0
	duplicateCount := 0
	for result := range results {
		acceptedCount += len(result.AcceptedEvents)
		duplicateCount += len(result.DuplicateEventIDs)
	}
	for createError := range errors {
		if createError != nil {
			t.Fatalf("concurrent duplicate submission failed: %v", createError)
		}
	}
	if acceptedCount != 1 || duplicateCount != 1 {
		t.Fatalf("expected one accepted and one duplicate event, got accepted=%d duplicate=%d", acceptedCount, duplicateCount)
	}
}

func TestDayTrackingService_CreateEvents_AppendsAfterSavedActualBlocks(t *testing.T) {
	// Arrange
	database := setupTestDB(t)
	user := createTestUser(t, database, "incremental-blocks", "password123")
	firstCategory, _ := NewCategoryRepository(database).Create(context.Background(), CategoryInput{Name: "First", Color: "#FF5733"}, user.ID)
	secondCategory, _ := NewCategoryRepository(database).Create(context.Background(), CategoryInput{Name: "Second", Color: "#3357FF"}, user.ID)
	record, err := newDayTrackingServiceForTest(database).CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-07-07"))
	if err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	_, err = newDayTrackingServiceForTest(database).ReplaceActualBlocks(context.Background(), record.ID, user.ID, []ActualBlockInput{{
		CategoryID:      &firstCategory.ID,
		BlockType:       "actual",
		StartTime:       mustScheduleTime("09:00:00"),
		DurationMinutes: 60,
	}}, nil)
	if err != nil {
		t.Fatalf("failed to seed actual block: %v", err)
	}

	// Act
	_, _, err = newDayTrackingServiceForTest(database).CreateEvents(context.Background(), record.ID, user.ID, []DayEventInput{
		{EventType: "transition", CategoryID: &secondCategory.ID, OccurredAt: parseTime("2026-07-07T10:05:00Z")},
		{EventType: "confirmation", CategoryID: &secondCategory.ID, OccurredAt: parseTime("2026-07-07T10:15:00Z")},
	})

	// Assert
	if err != nil {
		t.Fatalf("CreateEvents failed: %v", err)
	}
	updatedRecord, err := newDayTrackingServiceForTest(database).FindByID(context.Background(), record.ID, user.ID)
	if err != nil {
		t.Fatalf("failed to reload day record: %v", err)
	}
	if len(updatedRecord.ActualBlocks) != 2 {
		t.Fatalf("expected saved block and one appended block, got %d", len(updatedRecord.ActualBlocks))
	}
	if updatedRecord.ActualBlocks[0].StartTime.Hour() != 9 || updatedRecord.ActualBlocks[0].StartTime.Minute() != 0 {
		t.Errorf("expected saved block to remain at 09:00:00, got %v", updatedRecord.ActualBlocks[0].StartTime)
	}
	if updatedRecord.ActualBlocks[1].DurationMinutes != 10 {
		t.Errorf("expected ten-minute appended block, got %d", updatedRecord.ActualBlocks[1].DurationMinutes)
	}
	if updatedRecord.ActualBlocks[1].CategoryID == nil || *updatedRecord.ActualBlocks[1].CategoryID != secondCategory.ID {
		t.Errorf("expected appended block to use category %d, got %v", secondCategory.ID, updatedRecord.ActualBlocks[1].CategoryID)
	}
}

func TestDayTrackingService_ReplaceActualBlocks_RollsBackPartialBatch(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := NewCategoryRepository(db).Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	record, _ := newDayTrackingServiceForTest(db).CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-07-07"))
	_, err := newDayTrackingServiceForTest(db).ReplaceActualBlocks(context.Background(), record.ID, user.ID, []ActualBlockInput{
		{CategoryID: &category.ID, BlockType: "actual", StartTime: mustScheduleTime("09:00:00"), DurationMinutes: 60},
	}, nil)
	if err != nil {
		t.Fatalf("Failed to create initial actual block: %v", err)
	}
	invalidCategoryID := 99999

	// Act
	_, err = newDayTrackingServiceForTest(db).ReplaceActualBlocks(context.Background(), record.ID, user.ID, []ActualBlockInput{
		{CategoryID: &category.ID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 60},
		{CategoryID: &invalidCategoryID, BlockType: "actual", StartTime: mustScheduleTime("11:00:00"), DurationMinutes: 60},
	}, nil)

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
