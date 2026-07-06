package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteAccountHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DeleteAccountInput{Password: "password123"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.deleteAccountHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DeleteAccountResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Deleted {
		t.Error("Expected deleted=true in response")
	}

	// Verify user is actually deleted
	_, err := api.userRepo.FindByUsername(context.Background(), "testuser")
	if err == nil {
		t.Error("User should be deleted from database")
	}
}

func TestDeleteAccountHandler_WrongPassword(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DeleteAccountInput{Password: "wrongpassword"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.deleteAccountHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Verify user still exists
	foundUser, err := api.userRepo.FindByUsername(context.Background(), "testuser")
	if err != nil {
		t.Errorf("User should still exist: %v", err)
	}
	if foundUser.ID != user.ID {
		t.Error("Found different user")
	}
}

func TestDeleteAccountHandler_MissingPassword(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DeleteAccountInput{Password: ""}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.deleteAccountHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDeleteAccountHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	createTestUser(t, db, "testuser", "password123")

	reqBody := DeleteAccountInput{Password: "password123"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	api.deleteAccountHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestDeleteAccountHandler_InvalidJSON(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.deleteAccountHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDeleteAccountHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/account", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	api.deleteAccountHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}
