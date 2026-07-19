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

func TestGetCategoriesHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Personal", Color: "#33FF57"}, user.ID)

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getCategoriesHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response CategoriesResponse
	json.NewDecoder(w.Body).Decode(&response)
	if len(response.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(response.Categories))
	}
}

func TestGetCategoriesHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()

	// Act
	api.getCategoriesHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCreateCategoryHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := CategoryInput{Name: "Work", Color: "#FF5733"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.createCategoryHandler(w, req)

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var category BlockCategory
	json.NewDecoder(w.Body).Decode(&category)
	if category.Name != reqBody.Name {
		t.Errorf("Expected name '%s', got '%s'", reqBody.Name, category.Name)
	}
	if category.Color != reqBody.Color {
		t.Errorf("Expected color '%s', got '%s'", reqBody.Color, category.Color)
	}
}

func TestCreateCategoryHandler_MissingName(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := CategoryInput{Name: "", Color: "#FF5733"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.createCategoryHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateCategoryHandler_InvalidColor(t *testing.T) {
	testCases := []struct {
		name  string
		color string
	}{
		{"no hash", "FF5733"},
		{"too short", "#FF57"},
		{"too long", "#FF57334"},
		{"invalid chars", "#GGGGGG"},
		{"lowercase invalid", "#gggggg"},
		{"empty", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			db := setupTestDB(t)
			api := NewAPI(db, "test-secret", NewLogger("test"))
			user := createTestUser(t, db, "testuser_"+tc.name, "password123")

			reqBody := CategoryInput{Name: "Work", Color: tc.color}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := withUserID(context.Background(), user.ID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			// Act
			api.createCategoryHandler(w, req)

			// Assert
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for color '%s', got %d", tc.color, w.Code)
			}
		})
	}
}

func TestCreateCategoryHandler_ValidColors(t *testing.T) {
	validColors := []string{"#FF5733", "#000000", "#FFFFFF", "#AbCdEf", "#123456"}

	for _, color := range validColors {
		t.Run(color, func(t *testing.T) {
			// Arrange
			db := setupTestDB(t)
			api := NewAPI(db, "test-secret", NewLogger("test"))
			user := createTestUser(t, db, "testuser_"+color, "password123")

			reqBody := CategoryInput{Name: "Work", Color: color}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := withUserID(context.Background(), user.ID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			// Act
			api.createCategoryHandler(w, req)

			// Assert
			if w.Code != http.StatusCreated {
				t.Errorf("Expected status 201 for valid color '%s', got %d", color, w.Code)
			}
		})
	}
}

func TestUpdateCategoryHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	reqBody := CategoryInput{Name: "Deep Work", Color: "#0000FF"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.updateCategoryHandler(w, req, category.ID)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var updated BlockCategory
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != reqBody.Name {
		t.Errorf("Expected name '%s', got '%s'", reqBody.Name, updated.Name)
	}
	if updated.Color != reqBody.Color {
		t.Errorf("Expected color '%s', got '%s'", reqBody.Color, updated.Color)
	}
}

func TestUpdateCategoryHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := CategoryInput{Name: "Work", Color: "#FF5733"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/categories/99999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.updateCategoryHandler(w, req, 99999)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteCategoryHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	req := httptest.NewRequest(http.MethodDelete, "/categories", nil)

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.deleteCategoryHandler(w, req, category.ID)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response CategoryDeleteResponse
	json.NewDecoder(w.Body).Decode(&response)
	if !response.Deleted {
		t.Error("Expected deleted=true")
	}
	if response.ID != category.ID {
		t.Errorf("Expected ID %d, got %d", category.ID, response.ID)
	}
}

func TestDeleteCategoryHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/categories/99999", nil)

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.deleteCategoryHandler(w, req, 99999)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestCategoriesHandler_RouteDispatch(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	// Create a category to update/delete
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"GET list", http.MethodGet, "/categories", http.StatusOK},
		{"POST create", http.MethodPost, "/categories", http.StatusCreated},
		{"PUT update", http.MethodPut, "/categories/" + strconv.Itoa(category.ID), http.StatusOK},
		{"DELETE delete", http.MethodDelete, "/categories/" + strconv.Itoa(category.ID), http.StatusOK},
		{"PATCH not allowed", http.MethodPatch, "/categories", http.StatusMethodNotAllowed},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			var body *bytes.Reader
			if tc.method == http.MethodPost || tc.method == http.MethodPut {
				reqBody := CategoryInput{Name: "Test", Color: "#FF5733"}
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
			api.categoriesHandler(w, req)

			// Assert
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
