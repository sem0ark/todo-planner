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

func TestGetDayTemplatesHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{Name: "Weekday", SnapshotBlocks: []SnapshotBlockInput{}}, user.ID)
	api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{Name: "Weekend", SnapshotBlocks: []SnapshotBlockInput{}}, user.ID)

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getDayTemplatesHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DayTemplatesResponse
	json.NewDecoder(w.Body).Decode(&response)
	if len(response.Templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(response.Templates))
	}
}

func TestGetDayTemplatesHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	w := httptest.NewRecorder()

	// Act
	api.getDayTemplatesHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCreateDayTemplateHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	reqBody := DayTemplateInput{
		Name: "Weekday",
		SnapshotBlocks: []SnapshotBlockInput{
			{CategoryID: category.ID, StartTime: "09:00:00", DurationMinutes: 480},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.createDayTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var template DayTemplate
	json.NewDecoder(w.Body).Decode(&template)
	if template.Name != reqBody.Name {
		t.Errorf("Expected name '%s', got '%s'", reqBody.Name, template.Name)
	}
	if len(template.CurrentSnapshot.SnapshotBlocks) != 1 {
		t.Errorf("Expected 1 planned block, got %d", len(template.CurrentSnapshot.SnapshotBlocks))
	}
}

func TestCreateDayTemplateHandler_WithTemplateGroup(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	group, _ := api.templateGroupRepo.Create(context.Background(), TemplateGroupInput{Name: "Work"}, user.ID)

	reqBody := DayTemplateInput{
		Name:            "Weekday",
		TemplateGroupID: &group.ID,
		SnapshotBlocks:  []SnapshotBlockInput{},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.createDayTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var template DayTemplate
	json.NewDecoder(w.Body).Decode(&template)
	if template.TemplateGroupID == nil || *template.TemplateGroupID != group.ID {
		t.Errorf("Expected TemplateGroupID %d, got %v", group.ID, template.TemplateGroupID)
	}
}

func TestCreateDayTemplateHandler_MissingName(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DayTemplateInput{Name: "", SnapshotBlocks: []SnapshotBlockInput{}}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.createDayTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUpdateDayTemplateHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category1, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	category2, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Rest", Color: "#33FF57"}, user.ID)

	template, _ := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{
		Name: "Original",
		SnapshotBlocks: []SnapshotBlockInput{
			{CategoryID: category1.ID, StartTime: "09:00:00", DurationMinutes: 480},
		},
	}, user.ID)

	reqBody := DayTemplateInput{
		Name: "Updated",
		SnapshotBlocks: []SnapshotBlockInput{
			{CategoryID: category2.ID, StartTime: "10:00:00", DurationMinutes: 240},
			{CategoryID: category1.ID, StartTime: "14:00:00", DurationMinutes: 120},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.updateDayTemplateHandler(w, req, template.ID)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var updated DayTemplate
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != reqBody.Name {
		t.Errorf("Expected name '%s', got '%s'", reqBody.Name, updated.Name)
	}
	if len(updated.CurrentSnapshot.SnapshotBlocks) != 2 {
		t.Errorf("Expected 2 planned blocks, got %d", len(updated.CurrentSnapshot.SnapshotBlocks))
	}
}

func TestUpdateDayTemplateHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DayTemplateInput{Name: "Updated", SnapshotBlocks: []SnapshotBlockInput{}}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/templates/99999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.updateDayTemplateHandler(w, req, 99999)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteDayTemplateHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	template, _ := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{Name: "Weekday", SnapshotBlocks: []SnapshotBlockInput{}}, user.ID)

	req := httptest.NewRequest(http.MethodDelete, "/templates", nil)

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.deleteDayTemplateHandler(w, req, template.ID)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DayTemplateDeleteResponse
	json.NewDecoder(w.Body).Decode(&response)
	if !response.Deleted {
		t.Error("Expected deleted=true")
	}
	if response.ID != template.ID {
		t.Errorf("Expected ID %d, got %d", template.ID, response.ID)
	}
}

func TestDeleteDayTemplateHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/templates/99999", nil)

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.deleteDayTemplateHandler(w, req, 99999)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDayTemplatesHandler_RouteDispatch(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	template, _ := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{Name: "Weekday", SnapshotBlocks: []SnapshotBlockInput{}}, user.ID)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"GET list", http.MethodGet, "/templates", http.StatusOK},
		{"POST create", http.MethodPost, "/templates", http.StatusCreated},
		{"PUT update", http.MethodPut, "/templates/" + strconv.Itoa(template.ID), http.StatusOK},
		{"DELETE delete", http.MethodDelete, "/templates/" + strconv.Itoa(template.ID), http.StatusOK},
		{"PATCH not allowed", http.MethodPatch, "/templates", http.StatusMethodNotAllowed},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			var body *bytes.Reader
			if tc.method == http.MethodPost || tc.method == http.MethodPut {
				reqBody := DayTemplateInput{Name: "Test", SnapshotBlocks: []SnapshotBlockInput{}}
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
			api.dayTemplatesHandler(w, req)

			// Assert
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
