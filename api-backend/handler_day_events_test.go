package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestPostDayEventsHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	user := createTestUser(t, db, "eventuser", "password123")
	ctx := context.Background()

	// Create a day record
	var dayRecordID int
	err := db.QueryRow(ctx, `
		INSERT INTO day_records (user_id, calendar_date, review_status, created_at, updated_at)
		VALUES ($1, $2, 'Unreviewed', $3, $4)
		RETURNING id
	`, user.ID, "2026-07-01", time.Now(), time.Now()).Scan(&dayRecordID)
	if err != nil {
		t.Fatalf("Failed to create day record: %v", err)
	}

	// Create categories
	var cat1ID, cat2ID int
	err = db.QueryRow(ctx, `
		INSERT INTO block_categories (user_id, name, color, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, false, $4, $5)
		RETURNING id
	`, user.ID, "Work", "#FF0000", time.Now(), time.Now()).Scan(&cat1ID)
	if err != nil {
		t.Fatalf("Failed to create category 1: %v", err)
	}

	err = db.QueryRow(ctx, `
		INSERT INTO block_categories (user_id, name, color, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, false, $4, $5)
		RETURNING id
	`, user.ID, "Break", "#00FF00", time.Now(), time.Now()).Scan(&cat2ID)
	if err != nil {
		t.Fatalf("Failed to create category 2: %v", err)
	}

	now := time.Now()
	reqBody := DayEventsInput{
		Events: []DayEventInput{
			{
				EventType:  "confirmation",
				OccurredAt: now.Add(-2 * time.Hour),
			},
			{
				EventType:          "transition",
				OutgoingCategoryID: &cat1ID,
				IncomingCategoryID: &cat1ID,
				OccurredAt:         now.Add(-1 * time.Hour),
			},
			{
				EventType:          "transition",
				OutgoingCategoryID: &cat1ID,
				IncomingCategoryID: &cat2ID,
				OccurredAt:         now.Add(-30 * time.Minute),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records/"+strconv.Itoa(dayRecordID)+"/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response struct {
		CreatedEvents []DayEvent    `json:"created_events"`
		ActualBlocks  []ActualBlock `json:"actual_blocks"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.CreatedEvents) != 3 {
		t.Errorf("Expected 3 created events, got %d", len(response.CreatedEvents))
	}

	// Should create 2 actual blocks (confirmation is skipped in block computation)
	if len(response.ActualBlocks) != 2 {
		t.Errorf("Expected 2 actual blocks, got %d", len(response.ActualBlocks))
	}
}

func TestPostDayEventsHandler_Reviewed(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	user := createTestUser(t, db, "eventuser", "password123")
	ctx := context.Background()

	// Create a reviewed day record
	var dayRecordID int
	err := db.QueryRow(ctx, `
		INSERT INTO day_records (user_id, calendar_date, review_status, created_at, updated_at)
		VALUES ($1, $2, 'Reviewed', $3, $4)
		RETURNING id
	`, user.ID, "2026-07-01", time.Now(), time.Now()).Scan(&dayRecordID)
	if err != nil {
		t.Fatalf("Failed to create day record: %v", err)
	}

	reqBody := DayEventsInput{
		Events: []DayEventInput{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records/"+strconv.Itoa(dayRecordID)+"/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestPostDayEventsHandler_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	user := createTestUser(t, db, "eventuser", "password123")

	reqBody := DayEventsInput{
		Events: []DayEventInput{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records/99999/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestPostDayEventsHandler_InvalidEventType(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	user := createTestUser(t, db, "eventuser", "password123")
	ctx := context.Background()

	var dayRecordID int
	err := db.QueryRow(ctx, `
		INSERT INTO day_records (user_id, calendar_date, review_status, created_at, updated_at)
		VALUES ($1, $2, 'Unreviewed', $3, $4)
		RETURNING id
	`, user.ID, "2026-07-01", time.Now(), time.Now()).Scan(&dayRecordID)
	if err != nil {
		t.Fatalf("Failed to create day record: %v", err)
	}

	reqBody := DayEventsInput{
		Events: []DayEventInput{
			{
				EventType:  "invalid_type",
				OccurredAt: time.Now(),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records/"+strconv.Itoa(dayRecordID)+"/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPostDayEventsHandler_TransitionMissingCategories(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	user := createTestUser(t, db, "eventuser", "password123")
	ctx := context.Background()

	var dayRecordID int
	err := db.QueryRow(ctx, `
		INSERT INTO day_records (user_id, calendar_date, review_status, created_at, updated_at)
		VALUES ($1, $2, 'Unreviewed', $3, $4)
		RETURNING id
	`, user.ID, "2026-07-01", time.Now(), time.Now()).Scan(&dayRecordID)
	if err != nil {
		t.Fatalf("Failed to create day record: %v", err)
	}

	reqBody := DayEventsInput{
		Events: []DayEventInput{
			{
				EventType:  "transition",
				OccurredAt: time.Now(),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records/"+strconv.Itoa(dayRecordID)+"/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPostDayEventsHandler_NotChronological(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	user := createTestUser(t, db, "eventuser", "password123")
	ctx := context.Background()

	var dayRecordID int
	err := db.QueryRow(ctx, `
		INSERT INTO day_records (user_id, calendar_date, review_status, created_at, updated_at)
		VALUES ($1, $2, 'Unreviewed', $3, $4)
		RETURNING id
	`, user.ID, "2026-07-01", time.Now(), time.Now()).Scan(&dayRecordID)
	if err != nil {
		t.Fatalf("Failed to create day record: %v", err)
	}

	var catID int
	err = db.QueryRow(ctx, `
		INSERT INTO block_categories (user_id, name, color, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, false, $4, $5)
		RETURNING id
	`, user.ID, "Work", "#FF0000", time.Now(), time.Now()).Scan(&catID)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	now := time.Now()
	reqBody := DayEventsInput{
		Events: []DayEventInput{
			{
				EventType:          "transition",
				IncomingCategoryID: &catID,
				OccurredAt:         now,
			},
			{
				EventType:          "transition",
				IncomingCategoryID: &catID,
				OccurredAt:         now.Add(-1 * time.Hour), // Earlier than first
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records/"+strconv.Itoa(dayRecordID)+"/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPostDayEventsHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	reqBody := DayEventsInput{
		Events: []DayEventInput{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records/1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPostDayEventsHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	user := createTestUser(t, db, "eventuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/day-records/1/events", nil)
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestPostDayEventsHandler_ConfirmationWithCategories(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret", NewLogger("test"))

	user := createTestUser(t, db, "eventuser", "password123")
	ctx := context.Background()

	var dayRecordID int
	err := db.QueryRow(ctx, `
		INSERT INTO day_records (user_id, calendar_date, review_status, created_at, updated_at)
		VALUES ($1, $2, 'Unreviewed', $3, $4)
		RETURNING id
	`, user.ID, "2026-07-01", time.Now(), time.Now()).Scan(&dayRecordID)
	if err != nil {
		t.Fatalf("Failed to create day record: %v", err)
	}

	var catID int
	err = db.QueryRow(ctx, `
		INSERT INTO block_categories (user_id, name, color, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, false, $4, $5)
		RETURNING id
	`, user.ID, "Work", "#FF0000", time.Now(), time.Now()).Scan(&catID)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	reqBody := DayEventsInput{
		Events: []DayEventInput{
			{
				EventType:          "confirmation",
				IncomingCategoryID: &catID, // Should not have categories
				OccurredAt:         time.Now(),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/day-records/"+strconv.Itoa(dayRecordID)+"/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.postDayEventsHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
