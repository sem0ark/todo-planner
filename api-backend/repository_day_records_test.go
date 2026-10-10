package main

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDayRecordRepository_FindByID(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "day-record-find-by-id", "password123")
	repository := NewDayRecordRepository(database)
	createdRecord := createDayRecordForRepositoryTest(
		t,
		database,
		repository,
		user.ID,
		mustCalendarDate("2026-09-06"),
	)

	foundRecord, err := repository.FindByID(context.Background(), createdRecord.ID, user.ID)
	if err != nil {
		t.Fatalf("failed to find day record: %v", err)
	}
	if foundRecord.ID != createdRecord.ID || foundRecord.CalendarDate != createdRecord.CalendarDate {
		t.Fatalf("unexpected day record: %+v", foundRecord)
	}
}

func TestDayRecordRepository_FindByDate(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "day-record-find-by-date", "password123")
	repository := NewDayRecordRepository(database)
	calendarDate := mustCalendarDate("2026-09-06")
	createdRecord := createDayRecordForRepositoryTest(t, database, repository, user.ID, calendarDate)

	foundRecord, err := repository.FindByDate(context.Background(), user.ID, calendarDate)
	if err != nil {
		t.Fatalf("failed to find day record: %v", err)
	}
	if foundRecord.ID != createdRecord.ID || foundRecord.CalendarDate != calendarDate {
		t.Fatalf("unexpected day record: %+v", foundRecord)
	}
	if foundRecord.ActualBlocks != nil || foundRecord.Events != nil || foundRecord.SnapshotBlocks != nil {
		t.Fatalf("repository returned composed data: %+v", foundRecord)
	}
}

func TestDayRecordRepository_FindersReturnNotFoundForMissingRecords(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "day-record-not-found", "password123")
	repository := NewDayRecordRepository(database)
	calendarDate := mustCalendarDate("2026-09-06")

	_, findByDateError := repository.FindByDate(context.Background(), user.ID, calendarDate)
	_, findByIDError := repository.FindByID(context.Background(), 9999, user.ID)
	if !errors.Is(findByDateError, ErrDayRecordNotFound) {
		t.Fatalf("expected date lookup to return ErrDayRecordNotFound, got %v", findByDateError)
	}
	if !errors.Is(findByIDError, ErrDayRecordNotFound) {
		t.Fatalf("expected ID lookup to return ErrDayRecordNotFound, got %v", findByIDError)
	}
}

func TestDayRecordRepository_FindByDateRange(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "day-record-date-range", "password123")
	repository := NewDayRecordRepository(database)
	dateList := []CalendarDate{
		mustCalendarDate("2026-09-01"),
		mustCalendarDate("2026-09-05"),
		mustCalendarDate("2026-09-10"),
		mustCalendarDate("2026-09-15"),
	}
	for _, calendarDate := range dateList {
		createDayRecordForRepositoryTest(t, database, repository, user.ID, calendarDate)
	}

	foundRecords, err := repository.FindByDateRange(
		context.Background(),
		user.ID,
		mustCalendarDate("2026-09-05"),
		mustCalendarDate("2026-09-10"),
	)
	if err != nil {
		t.Fatalf("failed to find day records: %v", err)
	}
	if len(foundRecords) != 2 {
		t.Fatalf("expected two day records, got %d", len(foundRecords))
	}
	if foundRecords[0].CalendarDate != dateList[1] || foundRecords[1].CalendarDate != dateList[2] {
		t.Fatalf("unexpected date ordering: %s, %s", foundRecords[0].CalendarDate, foundRecords[1].CalendarDate)
	}
}

func TestDayRecordRepository_FindByDateRangeReturnsEmptySlice(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "day-record-empty-range", "password123")
	foundRecords, err := NewDayRecordRepository(database).FindByDateRange(
		context.Background(),
		user.ID,
		mustCalendarDate("2026-09-01"),
		mustCalendarDate("2026-09-30"),
	)
	if err != nil {
		t.Fatalf("failed to find day records: %v", err)
	}
	if len(foundRecords) != 0 {
		t.Fatalf("expected no day records, got %d", len(foundRecords))
	}
}

func TestDayRecordRepository_FindByIDDoesNotCrossUsers(t *testing.T) {
	database := setupTestDB(t)
	owner := createTestUser(t, database, "day-record-owner", "password123")
	otherUser := createTestUser(t, database, "day-record-other-user", "password123")
	repository := NewDayRecordRepository(database)
	createdRecord := createDayRecordForRepositoryTest(
		t,
		database,
		repository,
		owner.ID,
		mustCalendarDate("2026-09-06"),
	)

	_, err := repository.FindByID(context.Background(), createdRecord.ID, otherUser.ID)
	if !errors.Is(err, ErrDayRecordNotFound) {
		t.Fatalf("expected a user-scoped lookup to return ErrDayRecordNotFound, got %v", err)
	}
}

func TestDayRecordRepository_UpdateTemplateTx(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "day-record-update-template", "password123")
	repository := NewDayRecordRepository(database)
	calendarDate := calendarDateRelativeToToday(t, database, 1)
	createdRecord := createDayRecordForRepositoryTest(
		t,
		database,
		repository,
		user.ID,
		calendarDate,
	)
	template := createTestDayTemplate(t, database, user.ID, "Assigned template", nil)
	requestedTemplateID := template.ID
	requestedSnapshotID := template.CurrentSnapshot.ID

	transaction, err := database.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin template-update transaction: %v", err)
	}
	defer transaction.Rollback(context.Background())

	updatedRecordID, err := repository.UpdateTemplateTx(
		context.Background(),
		transaction,
		user.ID,
		calendarDate,
		&requestedTemplateID,
		&requestedSnapshotID,
	)
	if err != nil {
		t.Fatalf("failed to assign template to day record: %v", err)
	}
	if updatedRecordID != createdRecord.ID {
		t.Fatalf("expected updated record ID %d, got %d", createdRecord.ID, updatedRecordID)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("failed to commit template-update transaction: %v", err)
	}

	updatedRecord, err := repository.FindByID(context.Background(), createdRecord.ID, user.ID)
	if err != nil {
		t.Fatalf("failed to reload updated day record: %v", err)
	}
	if updatedRecord.DayTemplateID == nil || *updatedRecord.DayTemplateID != requestedTemplateID {
		t.Fatalf("expected template ID %d, got %v", requestedTemplateID, updatedRecord.DayTemplateID)
	}
	if updatedRecord.SnapshotID == nil || *updatedRecord.SnapshotID != requestedSnapshotID {
		t.Fatalf("expected snapshot ID %d, got %v", requestedSnapshotID, updatedRecord.SnapshotID)
	}
}

func TestDayRecordRepository_UpdateTemplateTxClearsTemplate(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "day-record-clear-template", "password123")
	repository := NewDayRecordRepository(database)
	calendarDate := calendarDateRelativeToToday(t, database, 1)
	createdRecord := createDayRecordForRepositoryTest(
		t,
		database,
		repository,
		user.ID,
		calendarDate,
	)
	template := createTestDayTemplate(t, database, user.ID, "Template to clear", nil)

	assignmentTransaction, err := database.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin template-assignment transaction: %v", err)
	}
	if _, err := repository.UpdateTemplateTx(
		context.Background(),
		assignmentTransaction,
		user.ID,
		calendarDate,
		&template.ID,
		&template.CurrentSnapshot.ID,
	); err != nil {
		assignmentTransaction.Rollback(context.Background())
		t.Fatalf("failed to assign template before clearing it: %v", err)
	}
	if err := assignmentTransaction.Commit(context.Background()); err != nil {
		t.Fatalf("failed to commit template-assignment transaction: %v", err)
	}

	transaction, err := database.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin template-clear transaction: %v", err)
	}
	defer transaction.Rollback(context.Background())

	updatedRecordID, err := repository.UpdateTemplateTx(
		context.Background(), transaction, user.ID, calendarDate, nil, nil,
	)
	if err != nil {
		t.Fatalf("failed to clear day-record template: %v", err)
	}
	if updatedRecordID != createdRecord.ID {
		t.Fatalf("expected updated record ID %d, got %d", createdRecord.ID, updatedRecordID)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("failed to commit template-clear transaction: %v", err)
	}
	clearedRecord, err := repository.FindByID(context.Background(), createdRecord.ID, user.ID)
	if err != nil {
		t.Fatalf("failed to reload cleared day record: %v", err)
	}
	if clearedRecord.DayTemplateID != nil || clearedRecord.SnapshotID != nil {
		t.Fatalf("expected template and snapshot references to be cleared, got %+v", clearedRecord)
	}
}

func TestDayRecordRepository_UpdateTemplateTxRejectsPastOrMissingRecords(t *testing.T) {
	testCases := []struct {
		name          string
		dayOffset     int
		createRecord  bool
		expectedError error
	}{
		{
			name:          "past record",
			dayOffset:     -1,
			createRecord:  true,
			expectedError: ErrDayRecordPast,
		},
		{
			name:          "missing record",
			dayOffset:     1,
			expectedError: ErrDayRecordNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database := setupTestDB(t)
			user := createTestUser(t, database, "day-record-update-"+testCase.name, "password123")
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

			transaction, err := database.Begin(context.Background())
			if err != nil {
				t.Fatalf("failed to begin template-update transaction: %v", err)
			}
			defer transaction.Rollback(context.Background())

			_, updateError := NewDayRecordRepository(database).UpdateTemplateTx(
				context.Background(), transaction, user.ID, calendarDate, nil, nil,
			)
			if !errors.Is(updateError, testCase.expectedError) {
				t.Fatalf("expected error %v, got %v", testCase.expectedError, updateError)
			}
		})
	}
}

func createDayRecordForRepositoryTest(
	t *testing.T,
	database *pgxpool.Pool,
	repository *DayRecordRepository,
	userID int,
	calendarDate CalendarDate,
) *DayRecord {
	t.Helper()
	transaction, err := database.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin day-record transaction: %v", err)
	}
	defer transaction.Rollback(context.Background())
	createdRecord, err := repository.CreateTx(
		context.Background(), transaction, userID, calendarDate, nil, nil,
	)
	if err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("failed to commit day-record transaction: %v", err)
	}
	return createdRecord
}
