package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestEmptyListSerialization verifies that empty slices serialize as [] not null
func TestEmptyListSerialization_Categories(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

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

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	categories, ok := response["categories"]
	if !ok {
		t.Fatal("Expected 'categories' field in response")
	}

	// Verify it's an empty array, not null
	categoriesArray, ok := categories.([]interface{})
	if !ok {
		t.Fatalf("Expected categories to be an array, got %T: %v", categories, categories)
	}
	if categoriesArray == nil {
		t.Error("Expected categories to be [], not null")
	}
	if len(categoriesArray) != 0 {
		t.Errorf("Expected empty array, got length %d", len(categoriesArray))
	}
}

func TestEmptyListSerialization_TemplateGroups(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

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

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	groups, ok := response["groups"]
	if !ok {
		t.Fatal("Expected 'groups' field in response")
	}

	// Verify it's an empty array, not null
	groupsArray, ok := groups.([]interface{})
	if !ok {
		t.Fatalf("Expected groups to be an array, got %T: %v", groups, groups)
	}
	if groupsArray == nil {
		t.Error("Expected groups to be [], not null")
	}
	if len(groupsArray) != 0 {
		t.Errorf("Expected empty array, got length %d", len(groupsArray))
	}
}

func TestEmptyListSerialization_Templates(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

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

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	templates, ok := response["templates"]
	if !ok {
		t.Fatal("Expected 'templates' field in response")
	}

	// Verify it's an empty array, not null
	templatesArray, ok := templates.([]interface{})
	if !ok {
		t.Fatalf("Expected templates to be an array, got %T: %v", templates, templates)
	}
	if templatesArray == nil {
		t.Error("Expected templates to be [], not null")
	}
	if len(templatesArray) != 0 {
		t.Errorf("Expected empty array, got length %d", len(templatesArray))
	}
}

func TestEmptyListSerialization_TemplateWithEmptySnapshotBlocks(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	// Create a template with no planned blocks
	template, err := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{
		Name:           "Empty Template",
		SnapshotBlocks: []SnapshotBlockInput{},
	}, user.ID)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

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

	var response struct {
		Templates []struct {
			ID   int           `json:"id"`
			Plan []interface{} `json:"plan"`
		} `json:"templates"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	if len(response.Templates) != 1 {
		t.Fatalf("Expected 1 template, got %d", len(response.Templates))
	}

	tmpl := response.Templates[0]
	if tmpl.ID != template.ID {
		t.Errorf("Expected template ID %d, got %d", template.ID, tmpl.ID)
	}

	// Verify snapshot_blocks is an empty array, not null.
	if tmpl.Plan == nil {
		t.Error("Expected plan to be [], not null")
	}
	if len(tmpl.Plan) != 0 {
		t.Errorf("Expected empty plan array, got length %d", len(tmpl.Plan))
	}
}

func TestEmptyListSerialization_DayRecordWithEmptyBlocks(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	// Create a day record with no snapshot (no template assigned)
	_, err := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-08")
	if err != nil {
		t.Fatalf("Failed to create day record: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/days?from=2026-07-08&to=2026-07-08", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.daysHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Days []struct {
			CalendarDate string `json:"calendar_date"`
			DayRecord    *struct {
				Plan   []interface{} `json:"plan"`
				Actual []interface{} `json:"actual"`
			} `json:"day_record"`
		} `json:"days"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	if len(response.Days) != 1 {
		t.Fatalf("Expected 1 day entry, got %d", len(response.Days))
	}

	record := response.Days[0]
	if record.CalendarDate != "2026-07-08" {
		t.Errorf("Expected calendar date 2026-07-08, got %s", record.CalendarDate)
	}

	// Verify snapshot is null when no template is assigned.
	if record.DayRecord == nil {
		t.Fatal("Expected day record")
	}

	// Verify actual_blocks is an empty array, not null
	if record.DayRecord.Actual == nil {
		t.Error("Expected actual to be [], not null")
	}
	if len(record.DayRecord.Actual) != 0 {
		t.Errorf("Expected empty actual array, got length %d", len(record.DayRecord.Actual))
	}
}

func TestEmptyListSerialization_ScheduleWithEmptyOverrides(t *testing.T) {
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

	var response struct {
		WeeklySchedule []interface{} `json:"weekly_schedule"`
		Overrides      []interface{} `json:"overrides"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	// Verify weekly_schedule is an array with 7 elements
	if response.WeeklySchedule == nil {
		t.Fatal("Expected weekly_schedule to be an array, not null")
	}
	if len(response.WeeklySchedule) != 7 {
		t.Errorf("Expected weekly_schedule to have 7 days, got %d", len(response.WeeklySchedule))
	}

	// Verify overrides is an empty array, not null
	if response.Overrides == nil {
		t.Error("Expected overrides to be [], not null")
	}
	if len(response.Overrides) != 0 {
		t.Errorf("Expected empty overrides array, got length %d", len(response.Overrides))
	}
}
