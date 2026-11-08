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

func TestGetDayHandlerMissingRecord(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "missing-day-user", "password123")
	request := httptest.NewRequest(http.MethodGet, "/days/2026-09-06", nil)
	request = request.WithContext(withUserID(context.Background(), user.ID))
	response := httptest.NewRecorder()

	api.daysHandler(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a missing day, got %d", response.Code)
	}
}

func TestInitCreatesAndLoadsDayRecord(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "init-user", "password123")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	requestBody := `{"device_id":` + strconv.Itoa(device.ID) + `,"calendar_date":"2026-09-06","timezone_offset_minutes":120}`
	request := httptest.NewRequest(http.MethodPost, "/init", strings.NewReader(requestBody))
	request = request.WithContext(withUserID(context.Background(), user.ID))
	response := httptest.NewRecorder()

	api.initHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		DayRecords []PublicDayRecord `json:"day_records"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode init response: %v", err)
	}
	if len(payload.DayRecords) != initPlanDays {
		t.Fatalf("expected %d initialized days, got %d", initPlanDays, len(payload.DayRecords))
	}
	if payload.DayRecords[0].CalendarDate != mustCalendarDate("2026-09-06") {
		t.Fatalf("expected first initialized date, got %s", payload.DayRecords[0].CalendarDate)
	}
	if payload.DayRecords[initPlanDays-1].CalendarDate != mustCalendarDate("2026-09-12") {
		t.Fatalf("expected last initialized date, got %s", payload.DayRecords[initPlanDays-1].CalendarDate)
	}
}

func TestInitRejectsForeignDevice(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	owner := createTestUser(t, database, "device-owner", "password123")
	otherUser := createTestUser(t, database, "device-other", "password123")
	device, err := api.deviceRepo.Create(context.Background(), owner.ID, "mobile")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	requestBody := `{"device_id":` + strconv.Itoa(device.ID) + `,"calendar_date":"2026-09-06","timezone_offset_minutes":120}`
	request := httptest.NewRequest(http.MethodPost, "/init", strings.NewReader(requestBody))
	request = request.WithContext(withUserID(context.Background(), otherUser.ID))
	response := httptest.NewRecorder()

	api.initHandler(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a foreign device, got %d", response.Code)
	}
}

func TestInitRequiresTimezoneOffset(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "init-no-offset-user", "password123")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	for name, body := range map[string]string{
		"missing":      `{"device_id":` + strconv.Itoa(device.ID) + `,"calendar_date":"2026-09-06"}`,
		"out of range": `{"device_id":` + strconv.Itoa(device.ID) + `,"calendar_date":"2026-09-06","timezone_offset_minutes":841}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/init", strings.NewReader(body))
			request = request.WithContext(withUserID(context.Background(), user.ID))
			response := httptest.NewRecorder()

			api.initHandler(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestInitAppliesOffsetOnlyToCreatedDays(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "init-offset-user", "password123")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	if _, err := api.dayTrackingService.CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-09-06"), -60); err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	body := `{"device_id":` + strconv.Itoa(device.ID) + `,"calendar_date":"2026-09-06","timezone_offset_minutes":120}`
	request := httptest.NewRequest(http.MethodPost, "/init", strings.NewReader(body))
	request = request.WithContext(withUserID(context.Background(), user.ID))
	response := httptest.NewRecorder()

	api.initHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		DayRecords []PublicDayRecord `json:"day_records"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.DayRecords[0].TimezoneOffsetMinutes != -60 {
		t.Fatalf("expected existing day to keep offset -60, got %d", payload.DayRecords[0].TimezoneOffsetMinutes)
	}
	for _, record := range payload.DayRecords[1:] {
		if record.TimezoneOffsetMinutes != 120 {
			t.Fatalf("expected created day %s to use offset 120, got %d", record.CalendarDate, record.TimezoneOffsetMinutes)
		}
	}
}
