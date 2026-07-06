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
	api := NewAPI(db, "test-secret")
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
	api := NewAPI(db, "test-secret")
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
	api := NewAPI(db, "test-secret")
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

func TestGetDayRecordsHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")

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
	api := NewAPI(db, "test-secret")
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
	api := NewAPI(db, "test-secret")
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
	api := NewAPI(db, "test-secret")
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
	api := NewAPI(db, "test-secret")
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
	api := NewAPI(db, "test-secret")
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
	api := NewAPI(db, "test-secret")
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
	api := NewAPI(db, "test-secret")

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
	api := NewAPI(db, "test-secret")
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

func TestPutDayRecordStatusHandler_Success_Reviewed(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	// Create test record
	repo := NewDayRecordRepository(db)
	ctx := context.Background()
	record, err := repo.Create(ctx, user.ID, "2026-07-07")
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	reqBody := DayRecordStatusInput{
		ReviewStatus: "Reviewed",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx = withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DayRecord
	json.NewDecoder(w.Body).Decode(&response)
	if response.ReviewStatus != "Reviewed" {
		t.Errorf("Expected review_status 'Reviewed', got '%s'", response.ReviewStatus)
	}
}

func TestPutDayRecordStatusHandler_Success_Ignored(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	// Create test record
	repo := NewDayRecordRepository(db)
	ctx := context.Background()
	record, err := repo.Create(ctx, user.ID, "2026-07-07")
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	reqBody := DayRecordStatusInput{
		ReviewStatus: "Ignored",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx = withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DayRecord
	json.NewDecoder(w.Body).Decode(&response)
	if response.ReviewStatus != "Ignored" {
		t.Errorf("Expected review_status 'Ignored', got '%s'", response.ReviewStatus)
	}
}

func TestPutDayRecordStatusHandler_InvalidStatus(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	// Create test record
	repo := NewDayRecordRepository(db)
	ctx := context.Background()
	record, err := repo.Create(ctx, user.ID, "2026-07-07")
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	reqBody := DayRecordStatusInput{
		ReviewStatus: "Invalid",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx = withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordStatusHandler_AlreadyReviewed(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	// Create test record and mark as reviewed
	repo := NewDayRecordRepository(db)
	ctx := context.Background()
	record, err := repo.Create(ctx, user.ID, "2026-07-07")
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}
	_, err = repo.UpdateStatus(ctx, record.ID, user.ID, "Reviewed")
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	reqBody := DayRecordStatusInput{
		ReviewStatus: "Ignored",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/"+strconv.Itoa(record.ID)+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx = withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordStatusHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DayRecordStatusInput{
		ReviewStatus: "Reviewed",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/9999/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestPutDayRecordStatusHandler_InvalidID(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	reqBody := DayRecordStatusInput{
		ReviewStatus: "Reviewed",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/invalid/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordStatusHandler_InvalidJSON(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPut, "/day-records/1/status", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPutDayRecordStatusHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")

	reqBody := DayRecordStatusInput{
		ReviewStatus: "Reviewed",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/day-records/1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPutDayRecordStatusHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	api := NewAPI(db, "test-secret")
	user := createTestUser(t, db, "testuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/day-records/1/status", nil)
	ctx := withUserID(context.Background(), user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	api.putDayRecordStatusHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}
