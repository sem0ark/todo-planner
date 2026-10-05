package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBackupRouterGetAndPost(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "backup-user", "password123")
	category := createTestCategory(t, database, user.ID, "Work", "#123456")
	calendarDate := mustCalendarDate("2026-09-06")
	if _, err := database.Exec(context.Background(), `
		INSERT INTO day_records (user_id, calendar_date) VALUES ($1, $2)
	`, user.ID, calendarDate); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(context.Background(), `
		INSERT INTO day_events (day_record_id, client_event_id, event_type, category_id,
			occurred_at, occurred_at_local)
		SELECT id, 'backup-event', 'transition', $1, $2, $3
		FROM day_records WHERE user_id = $4 AND calendar_date = $5
	`, category.ID, time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC),
		"2026-09-06T11:00:00+02:00", user.ID, calendarDate); err != nil {
		t.Fatal(err)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/backup", nil)
	getRequest = getRequest.WithContext(withUserID(context.Background(), user.ID))
	getResponse := httptest.NewRecorder()

	api.backupRouter(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected GET status 200, got %d", getResponse.Code)
	}
	var backup BackupData
	if err := json.NewDecoder(getResponse.Body).Decode(&backup); err != nil {
		t.Fatal(err)
	}
	if backup.Version != 1 || len(backup.Events) != 1 || backup.Events[0].CalendarDate != calendarDate {
		t.Fatalf("unexpected backup payload: %+v", backup)
	}

	timezoneOffsetMinutes := 120
	backupClientEventID := "backup-event-2"
	postPayload := BackupData{
		Version: 1,
		Days: []BackupDay{{
			CalendarDate:          calendarDate,
			TimezoneOffsetMinutes: &timezoneOffsetMinutes,
			TimezoneOffsetLocked:  true,
			Actual: []BackupActualBlock{{
				CategoryID:      &category.ID,
				BlockType:       "actual",
				StartTime:       "09:00:00",
				DurationMinutes: 60,
			}},
		}},
		Events: []BackupEvent{{
			CalendarDate:    calendarDate,
			ClientEventID:   &backupClientEventID,
			EventType:       "transition",
			CategoryID:      &category.ID,
			OccurredAt:      time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC),
			OccurredAtLocal: "2026-09-06T12:00:00+02:00",
		}},
	}
	postBody, err := json.Marshal(postPayload)
	if err != nil {
		t.Fatal(err)
	}
	postRequest := httptest.NewRequest(http.MethodPost, "/backup", bytes.NewReader(postBody))
	postRequest = postRequest.WithContext(withUserID(context.Background(), user.ID))
	postResponse := httptest.NewRecorder()

	api.backupRouter(postResponse, postRequest)

	if postResponse.Code != http.StatusOK {
		t.Fatalf("expected POST status 200, got %d: %s", postResponse.Code, postResponse.Body.String())
	}
	var importResponse struct {
		ImportedEvents int `json:"imported_events"`
	}
	if err := json.NewDecoder(postResponse.Body).Decode(&importResponse); err != nil {
		t.Fatal(err)
	}
	if importResponse.ImportedEvents != 1 {
		t.Fatalf("expected one imported event, got %d", importResponse.ImportedEvents)
	}
	var timezoneOffset int
	var timezoneLocked bool
	if err := database.QueryRow(context.Background(), `
		SELECT timezone_offset_minutes, timezone_offset_locked
		FROM day_records WHERE user_id = $1 AND calendar_date = $2
	`, user.ID, calendarDate).Scan(&timezoneOffset, &timezoneLocked); err != nil {
		t.Fatal(err)
	}
	if timezoneOffset != 120 || !timezoneLocked {
		t.Fatalf("expected imported timezone metadata, got %d/%t", timezoneOffset, timezoneLocked)
	}
	var actualBlockCount int
	if err := database.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM actual_blocks
		JOIN day_records ON day_records.id = actual_blocks.day_record_id
		WHERE day_records.user_id = $1 AND day_records.calendar_date = $2
	`, user.ID, calendarDate).Scan(&actualBlockCount); err != nil {
		t.Fatal(err)
	}
	if actualBlockCount != 1 {
		t.Fatalf("expected one imported actual block, got %d", actualBlockCount)
	}
}

func TestBackupRouterRejectsInvalidBackup(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	request := httptest.NewRequest(http.MethodPost, "/backup", bytes.NewBufferString(`{"version":2}`))
	request = request.WithContext(withUserID(context.Background(), 1))
	response := httptest.NewRecorder()

	api.backupRouter(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}
