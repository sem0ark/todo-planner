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

func TestGetDayRecordsHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	// Create test records
	repo := NewDayRecordRepository(db)
	ctx := context.Background()
	_, err := repo.Create(ctx, user.ID, "2026-07-01")
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}
	_, err = repo.Create(ctx, user.ID, "2026-07-05")
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/day-records?from=2026-07-01&to=2026-07-05", nil)
	ctx = withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getDayRecordsHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DayRecordsResponse
	json.NewDecoder(w.Body).Decode(&response)
	if len(response.DayRecords) != 2 {
		t.Errorf("Expected 2 day records, got %d", len(response.DayRecords))
	}
}

func TestGetDayRecordsHandler_MissingQueryParams(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/day-records?from=2026-07-01", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getDayRecordsHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetDayRecordsHandler_InvalidDateFormat(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/day-records?from=invalid&to=2026-07-05", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getDayRecordsHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetDayRecordsHandler_DateRangeTooLarge(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/day-records?from=2026-01-01&to=2026-02-02", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getDayRecordsHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetDayRecordsHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	req := httptest.NewRequest(http.MethodGet, "/day-records?from=2026-07-01&to=2026-07-05", nil)
	w := httptest.NewRecorder()

	// Act
	api.getDayRecordsHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestGetDayRecordsHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPost, "/day-records?from=2026-07-01&to=2026-07-05", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.getDayRecordsHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestPostDayRecordHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DayRecordInput{
		CalendarDate: "2026-07-07",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.postDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response DayRecord
	json.NewDecoder(w.Body).Decode(&response)
	if response.CalendarDate != "2026-07-07" {
		t.Errorf("Expected calendar_date '2026-07-07', got '%s'", response.CalendarDate)
	}
	if response.ReviewStatus != "Unreviewed" {
		t.Errorf("Expected review_status 'Unreviewed', got '%s'", response.ReviewStatus)
	}
}

func TestPostDayRecordHandler_DuplicateDate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	// Create first record
	repo := NewDayRecordRepository(db)
	ctx := context.Background()
	_, err := repo.Create(ctx, user.ID, "2026-07-07")
	if err != nil {
		t.Fatalf("Failed to create first record: %v", err)
	}

	reqBody := DayRecordInput{
		CalendarDate: "2026-07-07",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx = withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.postDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestPostDayRecordHandler_MissingDate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DayRecordInput{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.postDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPostDayRecordHandler_InvalidDateFormat(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DayRecordInput{
		CalendarDate: "07/07/2026",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.postDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPostDayRecordHandler_InvalidJSON(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPost, "/day-records", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.postDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPostDayRecordHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	reqBody := DayRecordInput{
		CalendarDate: "2026-07-07",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	api.postDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPostDayRecordHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/day-records", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.postDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &category.ID, BlockType: "actual", StartTime: "09:00:00", DurationMinutes: 120},
			{BlockType: "blank", StartTime: "11:00:00", DurationMinutes: 60},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var response struct {
		ActualBlocks []ActualBlock `json:"actual_blocks"`
	}
	json.NewDecoder(w.Body).Decode(&response)
	if len(response.ActualBlocks) != 2 {
		t.Errorf("Expected 2 actual blocks, got %d", len(response.ActualBlocks))
	}
}

func TestPutDayRecordHandler_InvalidBlockType(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &category.ID, BlockType: "invalid", StartTime: "09:00:00", DurationMinutes: 120},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_ActualBlockMissingCategory(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{BlockType: "actual", StartTime: "09:00:00", DurationMinutes: 120},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_BlankBlockWithCategory(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &category.ID, BlockType: "blank", StartTime: "09:00:00", DurationMinutes: 120},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_InvalidStartTime(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &category.ID, BlockType: "actual", StartTime: "invalid", DurationMinutes: 120},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_DurationTooShort(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &category.ID, BlockType: "actual", StartTime: "09:00:00", DurationMinutes: 15},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_BlockExtendsPastMidnight(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &category.ID, BlockType: "actual", StartTime: "23:30:00", DurationMinutes: 60},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_InvalidCategoryID(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	invalidCategoryID := 99999
	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &invalidCategoryID, BlockType: "actual", StartTime: "09:00:00", DurationMinutes: 120},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &category.ID, BlockType: "actual", StartTime: "09:00:00", DurationMinutes: 120},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/99999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")
	category, _ := api.categoryRepo.Create(context.Background(), CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordBlocksInput{
		ActualBlocks: []ActualBlockInput{
			{CategoryID: &category.ID, BlockType: "actual", StartTime: "09:00:00", DurationMinutes: 120},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPutDayRecordHandler_InvalidJSON(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordTemplateHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	template, _ := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{Name: "Weekday", SnapshotBlocks: []SnapshotBlockInput{}}, user.ID)
	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2099-01-01")

	reqBody := DayRecordTemplateInput{
		DayTemplateID: &template.ID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var response DayRecord
	json.NewDecoder(w.Body).Decode(&response)
	if response.DayTemplateID == nil || *response.DayTemplateID != template.ID {
		t.Errorf("Expected DayTemplateID %d, got %v", template.ID, response.DayTemplateID)
	}
}

func TestPutDayRecordTemplateHandler_RemoveTemplate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	template, _ := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{Name: "Weekday", SnapshotBlocks: []SnapshotBlockInput{}}, user.ID)
	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2099-01-01")

	// First assign a template
	api.dayRecordRepo.UpdateTemplate(context.Background(), user.ID, record.ID, &template.ID)

	reqBody := DayRecordTemplateInput{
		DayTemplateID: nil,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DayRecord
	json.NewDecoder(w.Body).Decode(&response)
	if response.DayTemplateID != nil {
		t.Errorf("Expected DayTemplateID to be nil, got %v", response.DayTemplateID)
	}
}

func TestPutDayRecordTemplateHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	template, _ := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{Name: "Weekday", SnapshotBlocks: []SnapshotBlockInput{}}, user.ID)

	reqBody := DayRecordTemplateInput{
		DayTemplateID: &template.ID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/99999/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestPutDayRecordTemplateHandler_TemplateNotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2099-01-01")

	invalidTemplateID := 99999
	reqBody := DayRecordTemplateInput{
		DayTemplateID: &invalidTemplateID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestPutDayRecordTemplateHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	template, _ := api.dayTemplateRepo.Create(context.Background(), DayTemplateInput{Name: "Weekday", SnapshotBlocks: []SnapshotBlockInput{}}, user.ID)
	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	reqBody := DayRecordTemplateInput{
		DayTemplateID: &template.ID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPutDayRecordTemplateHandler_InvalidJSON(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret", NewLogger("test"))
	user := createTestUser(t, db, "testuser", "password123")

	record, _ := api.dayRecordRepo.Create(context.Background(), user.ID, "2026-07-07")

	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/template", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordTemplateHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
