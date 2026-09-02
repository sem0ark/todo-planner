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

func TestGetScheduleHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ScheduleResponse
	json.NewDecoder(w.Body).Decode(&response)
	if len(response.WeeklySchedule) != 7 {
		t.Errorf("Expected 7 days in weekly schedule, got %d", len(response.WeeklySchedule))
	}
	if response.Overrides == nil {
		t.Error("Expected overrides array, got nil")
	}
}

func TestGetScheduleHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	w := httptest.NewRecorder()

	// Act
	api.getScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestGetScheduleHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPost, "/schedule", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestGetTodayScheduleHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	template := createTestDayTemplate(t, db, user.ID, "Today", nil)
	today := time.Now()
	dayOfWeek := (int(today.Weekday()) + 6) % 7
	_, err := db.Exec(context.Background(), `
		INSERT INTO weekly_schedule (user_id, day_of_week, day_template_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, day_of_week) DO UPDATE SET day_template_id = EXCLUDED.day_template_id
	`, user.ID, dayOfWeek, template.ID)
	if err != nil {
		t.Fatalf("failed to seed today's schedule: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/schedule/today", nil)
	req = req.WithContext(withUserID(context.Background(), user.ID))
	w := httptest.NewRecorder()

	// Act
	api.getTodayScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
	var response TodayScheduleResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.CalendarDate != today.Format("2006-01-02") {
		t.Errorf("Expected today's date, got %s", response.CalendarDate)
	}
	if response.DayTemplateID == nil || *response.DayTemplateID != template.ID {
		t.Fatalf("Expected template ID %d, got %v", template.ID, response.DayTemplateID)
	}
	if response.Template == nil || response.Template.ID != template.ID {
		t.Errorf("Expected resolved template in response")
	}
}

func TestGetTodayScheduleHandler_OverrideTakesPrecedence(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	weekly := createTestDayTemplate(t, db, user.ID, "Weekly", nil)
	override := createTestDayTemplate(t, db, user.ID, "Override", nil)
	today := time.Now().Format("2006-01-02")
	dayOfWeek := (int(time.Now().Weekday()) + 6) % 7
	_, err := db.Exec(context.Background(), `
		INSERT INTO weekly_schedule (user_id, day_of_week, day_template_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, day_of_week) DO UPDATE SET day_template_id = EXCLUDED.day_template_id
	`, user.ID, dayOfWeek, weekly.ID)
	if err != nil {
		t.Fatalf("failed to seed weekly schedule: %v", err)
	}
	_, err = db.Exec(context.Background(), `
		INSERT INTO schedule_overrides (user_id, calendar_date, day_template_id)
		VALUES ($1, $2, $3)
	`, user.ID, today, override.ID)
	if err != nil {
		t.Fatalf("failed to seed schedule override: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/schedule/today", nil)
	req = req.WithContext(withUserID(context.Background(), user.ID))
	w := httptest.NewRecorder()

	// Act
	api.getTodayScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
	var response TodayScheduleResponse
	json.NewDecoder(w.Body).Decode(&response)
	if response.DayTemplateID == nil || *response.DayTemplateID != override.ID {
		t.Errorf("Expected override template ID %d, got %v", override.ID, response.DayTemplateID)
	}
}

func TestGetTodayScheduleHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	req := httptest.NewRequest(http.MethodGet, "/schedule/today", nil)
	w := httptest.NewRecorder()

	// Act
	api.getTodayScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPutWeeklyScheduleHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	template1 := createTestDayTemplate(t, db, user.ID, "Workday", nil)
	template2 := createTestDayTemplate(t, db, user.ID, "Weekend", nil)

	reqBody := WeeklyScheduleInput{
		WeeklySchedule: []WeeklyScheduleEntry{
			{DayOfWeek: 0, DayTemplateID: &template1.ID},
			{DayOfWeek: 1, DayTemplateID: &template2.ID},
			{DayOfWeek: 2, DayTemplateID: nil},
			{DayOfWeek: 3, DayTemplateID: &template1.ID},
			{DayOfWeek: 4, DayTemplateID: &template1.ID},
			{DayOfWeek: 5, DayTemplateID: nil},
			{DayOfWeek: 6, DayTemplateID: nil},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/weekly", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putWeeklyScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response WeeklyScheduleResponse
	json.NewDecoder(w.Body).Decode(&response)
	if len(response.WeeklySchedule) != 7 {
		t.Errorf("Expected 7 days, got %d", len(response.WeeklySchedule))
	}
	if response.WeeklySchedule[0].DayTemplateID == nil || *response.WeeklySchedule[0].DayTemplateID != template1.ID {
		t.Error("Day 0 should have template ID 1")
	}
}

func TestPutWeeklyScheduleHandler_NotEnoughDays(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := WeeklyScheduleInput{
		WeeklySchedule: []WeeklyScheduleEntry{
			{DayOfWeek: 0, DayTemplateID: nil},
			{DayOfWeek: 1, DayTemplateID: nil},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/weekly", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putWeeklyScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutWeeklyScheduleHandler_DuplicateDays(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := WeeklyScheduleInput{
		WeeklySchedule: []WeeklyScheduleEntry{
			{DayOfWeek: 0, DayTemplateID: nil},
			{DayOfWeek: 0, DayTemplateID: nil},
			{DayOfWeek: 2, DayTemplateID: nil},
			{DayOfWeek: 3, DayTemplateID: nil},
			{DayOfWeek: 4, DayTemplateID: nil},
			{DayOfWeek: 5, DayTemplateID: nil},
			{DayOfWeek: 6, DayTemplateID: nil},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/weekly", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putWeeklyScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutWeeklyScheduleHandler_InvalidJSON(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPut, "/schedule/weekly", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putWeeklyScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutWeeklyScheduleHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	reqBody := WeeklyScheduleInput{
		WeeklySchedule: []WeeklyScheduleEntry{
			{DayOfWeek: 0, DayTemplateID: nil},
			{DayOfWeek: 1, DayTemplateID: nil},
			{DayOfWeek: 2, DayTemplateID: nil},
			{DayOfWeek: 3, DayTemplateID: nil},
			{DayOfWeek: 4, DayTemplateID: nil},
			{DayOfWeek: 5, DayTemplateID: nil},
			{DayOfWeek: 6, DayTemplateID: nil},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/weekly", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	api.putWeeklyScheduleHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPutScheduleOverrideHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	template := createTestDayTemplate(t, db, user.ID, "Holiday", nil)

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	reqBody := ScheduleOverrideInput{
		DayTemplateID: &template.ID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/overrides/"+tomorrow, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putScheduleOverrideHandler(w, req, tomorrow)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ScheduleOverride
	json.NewDecoder(w.Body).Decode(&response)
	if response.CalendarDate != tomorrow {
		t.Errorf("Expected date %s, got %s", tomorrow, response.CalendarDate)
	}
	if response.DayTemplateID == nil || *response.DayTemplateID != template.ID {
		t.Error("Expected template ID to be set")
	}
}

func TestPutScheduleOverrideHandler_Delete(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	reqBody := ScheduleOverrideInput{
		DayTemplateID: nil,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/overrides/"+tomorrow, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putScheduleOverrideHandler(w, req, tomorrow)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ScheduleOverride
	json.NewDecoder(w.Body).Decode(&response)
	if response.DayTemplateID != nil {
		t.Error("Expected template ID to be nil (deleted)")
	}
}

func TestPutScheduleOverrideHandler_InvalidDateFormat(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	invalidDate := "2024-13-01" // Invalid month

	reqBody := ScheduleOverrideInput{
		DayTemplateID: nil,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/overrides/"+invalidDate, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putScheduleOverrideHandler(w, req, invalidDate)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutScheduleOverrideHandler_PastDate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	reqBody := ScheduleOverrideInput{
		DayTemplateID: nil,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/overrides/"+yesterday, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putScheduleOverrideHandler(w, req, yesterday)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutScheduleOverrideHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	reqBody := ScheduleOverrideInput{
		DayTemplateID: nil,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/schedule/overrides/"+tomorrow, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	api.putScheduleOverrideHandler(w, req, tomorrow)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestScheduleHandler_RouteDispatch(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	testCases := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
	}{
		{"GET schedule", http.MethodGet, "/schedule", nil, http.StatusOK},
		{"GET today's schedule", http.MethodGet, "/schedule/today", nil, http.StatusOK},
		{"PUT weekly", http.MethodPut, "/schedule/weekly", WeeklyScheduleInput{
			WeeklySchedule: []WeeklyScheduleEntry{
				{DayOfWeek: 0}, {DayOfWeek: 1}, {DayOfWeek: 2},
				{DayOfWeek: 3}, {DayOfWeek: 4}, {DayOfWeek: 5}, {DayOfWeek: 6},
			},
		}, http.StatusOK},
		{"PUT override", http.MethodPut, "/schedule/overrides/" + tomorrow, ScheduleOverrideInput{DayTemplateID: nil}, http.StatusOK},
		{"unknown path", http.MethodGet, "/schedule/unknown", nil, http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			var body *bytes.Reader
			if tc.body != nil {
				jsonBody, _ := json.Marshal(tc.body)
				body = bytes.NewReader(jsonBody)
			} else {
				body = bytes.NewReader([]byte{})
			}

			req := httptest.NewRequest(tc.method, tc.path, body)
			req.Header.Set("Content-Type", "application/json")

			ctx := withUserID(context.Background(), user.ID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			// Act
			api.scheduleHandler(w, req)

			// Assert
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
