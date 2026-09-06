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
	requestBody := `{"device_id":` + strconv.Itoa(device.ID) + `,"calendar_date":"2026-09-06"}`
	request := httptest.NewRequest(http.MethodPost, "/init", strings.NewReader(requestBody))
	request = request.WithContext(withUserID(context.Background(), user.ID))
	response := httptest.NewRecorder()

	api.initHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		DayRecord publicDayRecord `json:"day_record"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode init response: %v", err)
	}
	if payload.DayRecord.CalendarDate != "2026-09-06" {
		t.Fatalf("expected initialized date, got %s", payload.DayRecord.CalendarDate)
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
	requestBody := `{"device_id":` + strconv.Itoa(device.ID) + `,"calendar_date":"2026-09-06"}`
	request := httptest.NewRequest(http.MethodPost, "/init", strings.NewReader(requestBody))
	request = request.WithContext(withUserID(context.Background(), otherUser.ID))
	response := httptest.NewRecorder()

	api.initHandler(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a foreign device, got %d", response.Code)
	}
}
