package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func daysRequest(t *testing.T, api *API, userID int, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request = request.WithContext(withUserID(context.Background(), userID))
	response := httptest.NewRecorder()
	api.daysHandler(response, request)
	return response
}

func decodeDayResponse(t *testing.T, response *httptest.ResponseRecorder) PublicDayRecord {
	t.Helper()
	var record PublicDayRecord
	if err := json.Unmarshal(response.Body.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode day: %v; body=%s", err, response.Body.String())
	}
	return record
}

func TestPostDayTimezoneOffset(t *testing.T) {
	testCases := []struct {
		name           string
		body           string
		expectedCode   int
		expectedOffset int
	}{
		{name: "empty body defaults to zero", body: "", expectedCode: http.StatusCreated, expectedOffset: 0},
		{name: "empty object defaults to zero", body: "{}", expectedCode: http.StatusCreated, expectedOffset: 0},
		{name: "explicit offset", body: `{"timezone_offset_minutes":120}`, expectedCode: http.StatusCreated, expectedOffset: 120},
		{name: "negative offset", body: `{"timezone_offset_minutes":-300}`, expectedCode: http.StatusCreated, expectedOffset: -300},
		{name: "too large", body: `{"timezone_offset_minutes":841}`, expectedCode: http.StatusBadRequest},
		{name: "too small", body: `{"timezone_offset_minutes":-841}`, expectedCode: http.StatusBadRequest},
		{name: "unknown field", body: `{"client_offset_minutes":60}`, expectedCode: http.StatusBadRequest},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database := setupTestDB(t)
			api := NewAPI(database, "test-secret", NewLogger("test"))
			user := createTestUser(t, database, "post-day-offset", "password123")

			response := daysRequest(t, api, user.ID, http.MethodPost, "/days/2026-09-06", testCase.body)

			if response.Code != testCase.expectedCode {
				t.Fatalf("expected %d, got %d: %s", testCase.expectedCode, response.Code, response.Body.String())
			}
			if testCase.expectedCode != http.StatusCreated {
				if _, err := api.dayRecordRepo.FindByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06")); err == nil {
					t.Fatal("expected no record for an invalid request")
				}
				return
			}
			if record := decodeDayResponse(t, response); record.TimezoneOffsetMinutes != testCase.expectedOffset {
				t.Fatalf("expected offset %d, got %d", testCase.expectedOffset, record.TimezoneOffsetMinutes)
			}
		})
	}
}

func TestPutDayUpdatesOnlyOffset(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "put-day-offset", "password123")
	category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	date := mustCalendarDate("2026-09-06")
	if _, err := api.dayTrackingService.CreateDayRecord(context.Background(), user.ID, date, 0); err != nil {
		t.Fatalf("failed to create day: %v", err)
	}
	eventsResponse := postDateEventsRequest(t, api, user.ID, "2026-09-06", DayEventsInput{
		DeviceID: device.ID,
		Events: []DayEventInput{
			transitionInput("e1", &category.ID, "2026-09-06T08:00:00Z"),
			transitionInput("e2", &category.ID, "2026-09-06T09:00:00Z"),
		},
	})
	if eventsResponse.Code != http.StatusOK {
		t.Fatalf("events failed: %d %s", eventsResponse.Code, eventsResponse.Body.String())
	}
	before := decodeDayResponse(t, eventsResponse)

	response := daysRequest(t, api, user.ID, http.MethodPut, "/days/2026-09-06", `{"timezone_offset_minutes":-180}`)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	after := decodeDayResponse(t, response)
	if after.TimezoneOffsetMinutes != -180 {
		t.Fatalf("expected offset -180, got %d", after.TimezoneOffsetMinutes)
	}
	if len(after.Actual) != 1 || len(after.Events) != 2 ||
		after.Actual[0].StartAt != before.Actual[0].StartAt ||
		after.Actual[0].DurationMinutes != before.Actual[0].DurationMinutes {
		t.Fatalf("expected blocks and events unchanged, before=%+v after=%+v", before, after)
	}
}

func TestPutDayErrors(t *testing.T) {
	testCases := []struct {
		name         string
		createDay    bool
		body         string
		expectedCode int
	}{
		{name: "missing day", body: `{"timezone_offset_minutes":60}`, expectedCode: http.StatusNotFound},
		{name: "missing field", createDay: true, body: `{}`, expectedCode: http.StatusBadRequest},
		{name: "empty body", createDay: true, body: ``, expectedCode: http.StatusBadRequest},
		{name: "out of range", createDay: true, body: `{"timezone_offset_minutes":900}`, expectedCode: http.StatusBadRequest},
		{name: "invalid json", createDay: true, body: `nope`, expectedCode: http.StatusBadRequest},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database := setupTestDB(t)
			api := NewAPI(database, "test-secret", NewLogger("test"))
			user := createTestUser(t, database, "put-day-errors", "password123")
			if testCase.createDay {
				if _, err := api.dayTrackingService.CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-09-06"), 30); err != nil {
					t.Fatal(err)
				}
			}

			response := daysRequest(t, api, user.ID, http.MethodPut, "/days/2026-09-06", testCase.body)

			if response.Code != testCase.expectedCode {
				t.Fatalf("expected %d, got %d: %s", testCase.expectedCode, response.Code, response.Body.String())
			}
			if testCase.createDay {
				record, err := api.dayRecordRepo.FindByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06"))
				if err != nil || record.TimezoneOffsetMinutes != 30 {
					t.Fatalf("expected offset to remain 30, got %+v (%v)", record, err)
				}
			}
		})
	}
}

func TestPutDateBlocksRequiresExistingDay(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "put-blocks-missing-day", "password123")

	response := daysRequest(t, api, user.ID, http.MethodPut, "/days/2026-09-06/blocks", `{"actual":[]}`)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	if _, err := api.dayRecordRepo.FindByDate(context.Background(), user.ID, mustCalendarDate("2026-09-06")); err == nil {
		t.Fatal("expected no day record to be auto-created")
	}
}

func TestPutDateBlocksUsesStartAtInstants(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "put-blocks-start-at", "password123")
	category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
	if _, err := api.dayTrackingService.CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-09-06"), 120); err != nil {
		t.Fatal(err)
	}
	// A block crossing UTC midnight is valid: blocks are instants, not time-of-day.
	body := `{"actual":[{"category_id":` + strconv.Itoa(category.ID) + `,"block_type":"actual","start_at":"2026-09-06T23:30:00Z","duration_minutes":90}]}`

	response := daysRequest(t, api, user.ID, http.MethodPut, "/days/2026-09-06/blocks", body)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	record := decodeDayResponse(t, response)
	if len(record.Actual) != 1 || record.Actual[0].StartAt.Time().Format("2006-01-02T15:04:05Z") != "2026-09-06T23:30:00Z" ||
		record.Actual[0].DurationMinutes != 90 {
		t.Fatalf("unexpected actual blocks: %+v", record.Actual)
	}

	legacy := `{"actual":[{"block_type":"blank","start_time":"09:00:00","duration_minutes":30}]}`
	response = daysRequest(t, api, user.ID, http.MethodPut, "/days/2026-09-06/blocks", legacy)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected start_time to be rejected, got %d", response.Code)
	}
}

func TestPublicDayJSONShape(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "public-day-shape", "password123")
	category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatal(err)
	}
	if response := daysRequest(t, api, user.ID, http.MethodPost, "/days/2026-09-06", `{"timezone_offset_minutes":60}`); response.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", response.Code, response.Body.String())
	}
	postDateEventsRequest(t, api, user.ID, "2026-09-06", DayEventsInput{
		DeviceID: device.ID,
		Events: []DayEventInput{
			transitionInput("e1", &category.ID, "2026-09-06T08:00:00Z"),
			transitionInput("e2", &category.ID, "2026-09-06T09:00:00Z"),
		},
	})

	response := daysRequest(t, api, user.ID, http.MethodGet, "/days/2026-09-06", "")

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, removed := range []string{"timezone_offset_locked", "blocks_edited"} {
		if _, exists := raw[removed]; exists {
			t.Errorf("unexpected field %q in public day JSON", removed)
		}
	}
	if string(raw["timezone_offset_minutes"]) != "60" {
		t.Errorf("expected numeric offset 60, got %s", raw["timezone_offset_minutes"])
	}
	body := response.Body.String()
	if strings.Contains(body, "occurred_at_local") || strings.Contains(body, "corrected_at_local") || strings.Contains(body, `"start_time"`) {
		t.Errorf("removed fields leaked into public JSON: %s", body)
	}
	var actual []map[string]any
	if err := json.Unmarshal(raw["actual"], &actual); err != nil || len(actual) != 1 {
		t.Fatalf("expected one actual block, got %s (%v)", raw["actual"], err)
	}
	if actual[0]["start_at"] != "2026-09-06T08:00:00Z" {
		t.Errorf("expected start_at 2026-09-06T08:00:00Z, got %v", actual[0]["start_at"])
	}
}
