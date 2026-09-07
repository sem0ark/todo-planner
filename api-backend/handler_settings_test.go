package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSettingsHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.getSettingsHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var settings PublicSettings
	if err := json.NewDecoder(w.Body).Decode(&settings); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if settings.DayBoundaryTime != mustScheduleTime("04:00:00") {
		t.Errorf("Expected default time '04:00:00', got '%v'", settings.DayBoundaryTime)
	}
}

func TestGetSettingsHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	// Act
	api.protectedHandler(api.getSettingsHandler)(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestGetSettingsHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPost, "/settings", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.settingsHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestPutSettingsHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	// Create initial settings
	if _, err := api.settingsRepo.Get(context.Background(), user.ID); err != nil {
		t.Fatalf("settings setup failed: %v", err)
	}

	reqBody := UserSettingsInput{DayBoundaryTime: mustScheduleTime("06:30:00")}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.putSettingsHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var settings PublicSettings
	if err := json.NewDecoder(w.Body).Decode(&settings); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if settings.DayBoundaryTime != mustScheduleTime("06:30:00") {
		t.Errorf("Expected time '06:30:00', got '%v'", settings.DayBoundaryTime)
	}
}

func TestPutSettingsHandler_InvalidTimeFormat(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	invalidTimes := []string{
		"25:00:00", // Invalid hour
		"12:60:00", // Invalid minute
		"12:30:60", // Invalid second
		"6:30:00",  // Missing leading zero
		"06:30",    // Missing seconds
		"invalid",  // Not a time
		"",         // Empty
	}

	for _, invalidTime := range invalidTimes {
		t.Run("InvalidTime_"+invalidTime, func(t *testing.T) {
			// Arrange
			body := fmt.Appendf(nil, `{"day_boundary_time":%q}`, invalidTime)
			req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := withUserID(context.Background(), user.ID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Act
			api.putSettingsHandler(w, req)

			// Assert
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for time '%s', got %d", invalidTime, w.Code)
			}
		})
	}
}

func TestPutSettingsHandler_ValidTimeFormats(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	if _, err := api.settingsRepo.Get(context.Background(), user.ID); err != nil {
		t.Fatalf("settings setup failed: %v", err)
	}

	validTimes := []string{
		"00:00:00",
		"04:00:00",
		"12:30:45",
		"23:59:59",
	}

	for _, validTime := range validTimes {
		t.Run("ValidTime_"+validTime, func(t *testing.T) {
			// Arrange
			reqBody := UserSettingsInput{DayBoundaryTime: mustScheduleTime(validTime)}
			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}
			req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := withUserID(context.Background(), user.ID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Act
			api.putSettingsHandler(w, req)

			// Assert
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for time '%s', got %d", validTime, w.Code)
			}

			var settings PublicSettings
			json.NewDecoder(w.Body).Decode(&settings)
			if settings.DayBoundaryTime != mustScheduleTime(validTime) {
				t.Errorf("Expected time '%s', got '%v'", validTime, settings.DayBoundaryTime)
			}
		})
	}
}

func TestPutSettingsHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	reqBody := UserSettingsInput{DayBoundaryTime: mustScheduleTime("06:30:00")}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	api.protectedHandler(api.putSettingsHandler)(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPutSettingsHandler_InvalidJSON(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.putSettingsHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutSettingsHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/settings", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.settingsHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}
