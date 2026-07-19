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

	groups, ok := response["template_groups"]
	if !ok {
		t.Fatal("Expected 'template_groups' field in response")
	}

	// Verify it's an empty array, not null
	groupsArray, ok := groups.([]interface{})
	if !ok {
		t.Fatalf("Expected template_groups to be an array, got %T: %v", groups, groups)
	}
	if groupsArray == nil {
		t.Error("Expected template_groups to be [], not null")
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

func TestEmptyListSerialization_TemplateWithEmptyPlannedBlocks(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	// Create a template with no planned blocks
	template, err := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{
		Name:          "Empty Template",
		PlannedBlocks: []PlannedBlockInput{},
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
			ID            int           `json:"id"`
			Name          string        `json:"name"`
			PlannedBlocks []interface{} `json:"planned_blocks"`
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

	// Verify planned_blocks is an empty array, not null
	if tmpl.PlannedBlocks == nil {
		t.Error("Expected planned_blocks to be [], not null")
	}
	if len(tmpl.PlannedBlocks) != 0 {
		t.Errorf("Expected empty planned_blocks array, got length %d", len(tmpl.PlannedBlocks))
	}
}

func TestEmptyListSerialization_DayRecordWithEmptyBlocks(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	// Create a day record with no snapshot (no template assigned)
	dayRecord, err := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-08")
	if err != nil {
		t.Fatalf("Failed to create day record: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/day-records?from=2026-07-08&to=2026-07-08", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getDayRecordsHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		DayRecords []struct {
			ID             int           `json:"id"`
			SnapshotID     *int          `json:"snapshot_id"`
			SnapshotBlocks []interface{} `json:"snapshot_blocks"`
			ActualBlocks   []interface{} `json:"actual_blocks"`
		} `json:"day_records"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	if len(response.DayRecords) != 1 {
		t.Fatalf("Expected 1 day record, got %d", len(response.DayRecords))
	}

	record := response.DayRecords[0]
	if record.ID != dayRecord.ID {
		t.Errorf("Expected day record ID %d, got %d", dayRecord.ID, record.ID)
	}

	// Verify snapshot_id is null
	if record.SnapshotID != nil {
		t.Errorf("Expected snapshot_id to be null, got %v", *record.SnapshotID)
	}

	// Verify snapshot_blocks is an empty array, not null
	if record.SnapshotBlocks == nil {
		t.Error("Expected snapshot_blocks to be [], not null")
	}
	if len(record.SnapshotBlocks) != 0 {
		t.Errorf("Expected empty snapshot_blocks array, got length %d", len(record.SnapshotBlocks))
	}

	// Verify actual_blocks is an empty array, not null
	if record.ActualBlocks == nil {
		t.Error("Expected actual_blocks to be [], not null")
	}
	if len(record.ActualBlocks) != 0 {
		t.Errorf("Expected empty actual_blocks array, got length %d", len(record.ActualBlocks))
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
