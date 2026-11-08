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
	template := createTestDayTemplate(t, database, user.ID, "Workday", nil)
	calendarDate := mustCalendarDate("2026-09-06")
	var dayRecordID int
	if err := database.QueryRow(context.Background(), `
		INSERT INTO day_records (user_id, day_template_id, calendar_date)
		VALUES ($1, $2, $3) RETURNING id
	`, user.ID, template.ID, calendarDate).Scan(&dayRecordID); err != nil {
		t.Fatal(err)
	}
	var snapshotID int
	if err := database.QueryRow(context.Background(), `
		INSERT INTO template_snapshots (day_template_id, user_id)
		VALUES ($1, $2) RETURNING id
	`, template.ID, user.ID).Scan(&snapshotID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(context.Background(), `
		UPDATE day_records SET snapshot_id = $1 WHERE id = $2
	`, snapshotID, dayRecordID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(context.Background(), `
		INSERT INTO snapshot_blocks (snapshot_id, category_id, start_time, duration_minutes)
		VALUES ($1, $2, '08:00:00', 60)
	`, snapshotID, category.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(context.Background(), `
		INSERT INTO actual_blocks (day_record_id, category_id, block_type, start_at, duration_minutes)
		VALUES ($1, $2, 'actual', '2026-09-06T09:00:00Z', 30)
	`, dayRecordID, category.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(context.Background(), `
		INSERT INTO day_events (day_record_id, client_event_id, event_type, category_id,
			occurred_at)
		SELECT id, 'backup-event', 'transition', $1, $2
		FROM day_records WHERE user_id = $3 AND calendar_date = $4
	`, category.ID, time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC),
		user.ID, calendarDate); err != nil {
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
	if backup.Version != 2 || len(backup.Events) != 1 || backup.Events[0].CalendarDate != calendarDate {
		t.Fatalf("unexpected backup payload: %+v", backup)
	}
	if len(backup.Days) != 1 || len(backup.Days[0].Plan) != 1 || len(backup.Days[0].Actual) != 1 {
		t.Fatalf("expected exported plan and actual blocks, got %+v", backup.Days)
	}

	timezoneOffsetMinutes := 120
	blockStartAt := time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC)
	backupClientEventID := "backup-event-2"
	unknownCategoryEventID := "backup-event-unknown-category"
	unknownCategoryID := 99999
	postPayload := BackupData{
		Version: 2,
		Days: []BackupDay{{
			CalendarDate:          calendarDate,
			TimezoneOffsetMinutes: &timezoneOffsetMinutes,
			Actual: []BackupActualBlock{{
				CategoryID:      &category.ID,
				BlockType:       "actual",
				StartAt:         &blockStartAt,
				DurationMinutes: 60,
			}},
		}},
		Events: []BackupEvent{{
			CalendarDate:  calendarDate,
			ClientEventID: &backupClientEventID,
			EventType:     "transition",
			CategoryID:    &category.ID,
			OccurredAt:    time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC),
		}, {
			CalendarDate:  calendarDate,
			ClientEventID: &unknownCategoryEventID,
			EventType:     "transition",
			CategoryID:    &unknownCategoryID,
			OccurredAt:    time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC),
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
	if importResponse.ImportedEvents != 2 {
		t.Fatalf("expected two imported events, got %d", importResponse.ImportedEvents)
	}
	var unknownCategory *int
	if err := database.QueryRow(context.Background(), `
		SELECT category_id FROM day_events
		JOIN day_records ON day_records.id = day_events.day_record_id
		WHERE day_records.user_id = $1 AND day_events.client_event_id = $2
	`, user.ID, unknownCategoryEventID).Scan(&unknownCategory); err != nil {
		t.Fatal(err)
	}
	if unknownCategory != nil {
		t.Fatalf("expected unknown historical category to be cleared, got %d", *unknownCategory)
	}
	var timezoneOffset int
	if err := database.QueryRow(context.Background(), `
		SELECT timezone_offset_minutes
		FROM day_records WHERE user_id = $1 AND calendar_date = $2
	`, user.ID, calendarDate).Scan(&timezoneOffset); err != nil {
		t.Fatal(err)
	}
	if timezoneOffset != 120 {
		t.Fatalf("expected imported timezone offset 120, got %d", timezoneOffset)
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
	var importedStartAt time.Time
	if err := database.QueryRow(context.Background(), `
		SELECT start_at FROM actual_blocks
		JOIN day_records ON day_records.id = actual_blocks.day_record_id
		WHERE day_records.user_id = $1 AND day_records.calendar_date = $2
	`, user.ID, calendarDate).Scan(&importedStartAt); err != nil {
		t.Fatal(err)
	}
	if !importedStartAt.Equal(blockStartAt) {
		t.Fatalf("expected imported start_at %v, got %v", blockStartAt, importedStartAt)
	}
}

func TestBackupRouterImportsVersion1UTCTimeOfDayBlocks(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "backup-v1-user", "password123")
	// Offset +120: the local day 2026-10-19 spans 2026-10-18T22:00Z to 2026-10-19T22:00Z.
	body := `{"version":1,"days":[{
		"calendar_date":"2026-10-19",
		"timezone_offset_minutes":120,
		"timezone_offset_locked":true,
		"actual":[
			{"block_type":"actual","start_time":"06:30:00","duration_minutes":30},
			{"block_type":"actual","start_time":"22:30:00","duration_minutes":30}
		],
		"created_at":"2026-10-19T00:00:00Z","updated_at":"2026-10-19T00:00:00Z"
	}]}`
	request := httptest.NewRequest(http.MethodPost, "/backup", bytes.NewBufferString(body))
	request = request.WithContext(withUserID(context.Background(), user.ID))
	response := httptest.NewRecorder()

	api.backupRouter(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	rows, err := database.Query(context.Background(), `
		SELECT start_at FROM actual_blocks
		JOIN day_records ON day_records.id = actual_blocks.day_record_id
		WHERE day_records.user_id = $1 ORDER BY start_at
	`, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var starts []time.Time
	for rows.Next() {
		var startAt time.Time
		if err := rows.Scan(&startAt); err != nil {
			t.Fatal(err)
		}
		starts = append(starts, startAt.UTC())
	}
	expected := []time.Time{parseTime("2026-10-18T22:30:00Z"), parseTime("2026-10-19T06:30:00Z")}
	if len(starts) != 2 || !starts[0].Equal(expected[0]) || !starts[1].Equal(expected[1]) {
		t.Fatalf("expected %v, got %v", expected, starts)
	}
}

func TestBackupActualBlockStartAtConvertsVersion1TimeOfDay(t *testing.T) {
	date := mustCalendarDate("2026-10-19")
	testCases := []struct {
		name     string
		offset   int
		start    string
		expected string
	}{
		{name: "inside the local day on the same UTC date", offset: 120, start: "06:30:00", expected: "2026-10-19T06:30:00Z"},
		{name: "before UTC midnight belongs to the previous UTC date", offset: 120, start: "22:30:00", expected: "2026-10-18T22:30:00Z"},
		{name: "negative offset", offset: -300, start: "02:00:00", expected: "2026-10-20T02:00:00Z"},
		{name: "zero offset", offset: 0, start: "00:00:00", expected: "2026-10-19T00:00:00Z"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			startAt, err := BackupActualBlock{StartTime: testCase.start}.startAt(date, testCase.offset)
			if err != nil {
				t.Fatal(err)
			}
			if !startAt.Equal(parseTime(testCase.expected)) {
				t.Fatalf("expected %s, got %s", testCase.expected, startAt)
			}
		})
	}
	explicit := parseTime("2026-10-19T12:00:00+02:00")
	startAt, err := BackupActualBlock{StartAt: &explicit, StartTime: "01:00:00"}.startAt(date, 120)
	if err != nil || !startAt.Equal(explicit) {
		t.Fatalf("expected explicit start_at to win, got %v (%v)", startAt, err)
	}
}

func TestBackupRouterRejectsInvalidBackup(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	request := httptest.NewRequest(http.MethodPost, "/backup", bytes.NewBufferString(`{"version":99}`))
	request = request.WithContext(withUserID(context.Background(), 1))
	response := httptest.NewRecorder()

	api.backupRouter(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}
