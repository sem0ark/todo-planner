package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestGetTemplateGroupsHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	api.templateGroupRepo.Create(context.Background(), TemplateGroupInput{Name: "Work"}, user.ID)
	api.templateGroupRepo.Create(context.Background(), TemplateGroupInput{Name: "Vacation"}, user.ID)

	req := httptest.NewRequest(http.MethodGet, "/template-groups", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getTemplateGroupsHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TemplateGroupsResponse
	json.NewDecoder(w.Body).Decode(&response)
	if len(response.TemplateGroups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(response.TemplateGroups))
	}
}

func TestGetTemplateGroupsHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/template-groups", nil)
	w := httptest.NewRecorder()

	// Act
	api.getTemplateGroupsHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCreateTemplateGroupHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := TemplateGroupInput{Name: "Full-Time Work"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/template-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.createTemplateGroupHandler(w, req)

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var group TemplateGroup
	json.NewDecoder(w.Body).Decode(&group)
	if group.Name != reqBody.Name {
		t.Errorf("Expected name '%s', got '%s'", reqBody.Name, group.Name)
	}
}

func TestCreateTemplateGroupHandler_MissingName(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := TemplateGroupInput{Name: ""}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/template-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.createTemplateGroupHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUpdateTemplateGroupHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	group, _ := api.templateGroupRepo.Create(context.Background(), TemplateGroupInput{Name: "Work"}, user.ID)

	reqBody := TemplateGroupInput{Name: "Full-Time Work"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/template-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.updateTemplateGroupHandler(w, req, group.ID)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var updated TemplateGroup
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != reqBody.Name {
		t.Errorf("Expected name '%s', got '%s'", reqBody.Name, updated.Name)
	}
}

func TestUpdateTemplateGroupHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := TemplateGroupInput{Name: "Work"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/template-groups/99999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.updateTemplateGroupHandler(w, req, 99999)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteTemplateGroupHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	group, _ := api.templateGroupRepo.Create(context.Background(), TemplateGroupInput{Name: "Work"}, user.ID)

	req := httptest.NewRequest(http.MethodDelete, "/template-groups", nil)

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.deleteTemplateGroupHandler(w, req, group.ID)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TemplateGroupDeleteResponse
	json.NewDecoder(w.Body).Decode(&response)
	if !response.Deleted {
		t.Error("Expected deleted=true")
	}
	if response.ID != group.ID {
		t.Errorf("Expected ID %d, got %d", group.ID, response.ID)
	}
}

func TestDeleteTemplateGroupHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/template-groups/99999", nil)

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.deleteTemplateGroupHandler(w, req, 99999)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestTemplateGroupsHandler_RouteDispatch(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	group, _ := api.templateGroupRepo.Create(context.Background(), TemplateGroupInput{Name: "Work"}, user.ID)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"GET list", http.MethodGet, "/template-groups", http.StatusOK},
		{"POST create", http.MethodPost, "/template-groups", http.StatusCreated},
		{"PUT update", http.MethodPut, "/template-groups/" + strconv.Itoa(group.ID), http.StatusOK},
		{"DELETE delete", http.MethodDelete, "/template-groups/" + strconv.Itoa(group.ID), http.StatusOK},
		{"PATCH not allowed", http.MethodPatch, "/template-groups", http.StatusMethodNotAllowed},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			var body *bytes.Reader
			if tc.method == http.MethodPost || tc.method == http.MethodPut {
				reqBody := TemplateGroupInput{Name: "Test"}
				jsonBody, _ := json.Marshal(reqBody)
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
			api.templateGroupsHandler(w, req)

			// Assert
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
