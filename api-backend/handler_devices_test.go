package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterDeviceHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DeviceInput{Platform: "mobile"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.registerDeviceHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DeviceResponse
	json.NewDecoder(w.Body).Decode(&response)
	if response.DeviceID == 0 {
		t.Error("Expected device_id to be set")
	}
	if response.RegisteredAt == "" {
		t.Error("Expected registered_at to be set")
	}
}

func TestRegisterDeviceHandler_ValidPlatforms(t *testing.T) {
	platforms := []string{"desktop", "mobile", "web"}

	for _, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			// Arrange
			db := setupTestDB(t)
			api := NewAPI(db, "test-secret", NewLogger("test"))
			user := createTestUser(t, db, "testuser_"+platform, "password123")

			reqBody := DeviceInput{Platform: platform}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := withUserID(context.Background(), user.ID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Act
			api.registerDeviceHandler(w, req)

			// Assert
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for platform %s, got %d", platform, w.Code)
			}
		})
	}
}

func TestRegisterDeviceHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	reqBody := DeviceInput{Platform: "mobile"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	api.registerDeviceHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRegisterDeviceHandler_InvalidPlatform(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DeviceInput{Platform: "invalid"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.registerDeviceHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRegisterDeviceHandler_MissingPlatform(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DeviceInput{Platform: ""}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.registerDeviceHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRegisterDeviceHandler_InvalidJSON(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.registerDeviceHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRegisterDeviceHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/devices", nil)

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.registerDeviceHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}
