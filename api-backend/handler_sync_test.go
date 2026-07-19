package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSyncHandler_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	user := createTestUser(t, db, "syncuser", "password123")
	ctx := context.Background()

	var deviceID int
	err := db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.ID, "mobile", "hash123", time.Now()).Scan(&deviceID)
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	reqBody := SyncRequest{
		DeviceID:   deviceID,
		LastSyncAt: nil,
		Changes: []ChangeLogEntry{
			{
				EntityType: "category",
				EntityID:   1,
				Operation:  "create",
				OccurredAt: time.Now(),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SyncResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.SyncedAt.IsZero() {
		t.Error("Expected synced_at to be set")
	}

	if len(response.Changes) != 0 {
		t.Errorf("Expected 0 remote changes, got %d", len(response.Changes))
	}

	var lastSyncAt time.Time
	err = db.QueryRow(ctx, "SELECT last_sync_at FROM devices WHERE id = $1", deviceID).Scan(&lastSyncAt)
	if err != nil {
		t.Fatalf("Failed to query last_sync_at: %v", err)
	}
	if lastSyncAt.IsZero() {
		t.Error("Expected last_sync_at to be updated")
	}
}

func TestSyncHandler_RemoteChanges(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	user := createTestUser(t, db, "syncuser", "password123")
	ctx := context.Background()

	var device1ID, device2ID int
	err := db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.ID, "mobile", "hash1", time.Now()).Scan(&device1ID)
	if err != nil {
		t.Fatalf("Failed to create device 1: %v", err)
	}

	err = db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.ID, "desktop", "hash2", time.Now()).Scan(&device2ID)
	if err != nil {
		t.Fatalf("Failed to create device 2: %v", err)
	}

	remoteChangeTime := time.Now().Add(-1 * time.Hour)
	_, err = db.Exec(ctx, `
		INSERT INTO change_log (device_id, user_id, entity_type, entity_id, operation, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, device2ID, user.ID, "category", 5, "update", remoteChangeTime)
	if err != nil {
		t.Fatalf("Failed to insert remote change: %v", err)
	}

	reqBody := SyncRequest{
		DeviceID:   device1ID,
		LastSyncAt: nil,
		Changes:    []ChangeLogEntry{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SyncResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Changes) != 1 {
		t.Fatalf("Expected 1 remote change, got %d", len(response.Changes))
	}

	change := response.Changes[0]
	if change.EntityType != "category" {
		t.Errorf("Expected entity_type 'category', got '%s'", change.EntityType)
	}
	if change.EntityID != 5 {
		t.Errorf("Expected entity_id 5, got %d", change.EntityID)
	}
	if change.Operation != "update" {
		t.Errorf("Expected operation 'update', got '%s'", change.Operation)
	}
}

func TestSyncHandler_LastSyncAtFilter(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	user := createTestUser(t, db, "syncuser", "password123")
	ctx := context.Background()

	var device1ID, device2ID int
	err := db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.ID, "mobile", "hash1", time.Now()).Scan(&device1ID)
	if err != nil {
		t.Fatalf("Failed to create device 1: %v", err)
	}

	err = db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.ID, "desktop", "hash2", time.Now()).Scan(&device2ID)
	if err != nil {
		t.Fatalf("Failed to create device 2: %v", err)
	}

	oldChangeTime := time.Now().Add(-2 * time.Hour)
	newChangeTime := time.Now().Add(-30 * time.Minute)
	lastSyncAt := time.Now().Add(-1 * time.Hour)

	_, err = db.Exec(ctx, `
		INSERT INTO change_log (device_id, user_id, entity_type, entity_id, operation, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, device2ID, user.ID, "category", 1, "create", oldChangeTime)
	if err != nil {
		t.Fatalf("Failed to insert old change: %v", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO change_log (device_id, user_id, entity_type, entity_id, operation, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, device2ID, user.ID, "category", 2, "update", newChangeTime)
	if err != nil {
		t.Fatalf("Failed to insert new change: %v", err)
	}

	reqBody := SyncRequest{
		DeviceID:   device1ID,
		LastSyncAt: &lastSyncAt,
		Changes:    []ChangeLogEntry{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SyncResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Changes) != 1 {
		t.Fatalf("Expected 1 remote change (only newer), got %d", len(response.Changes))
	}

	if response.Changes[0].EntityID != 2 {
		t.Errorf("Expected newer change with entity_id 2, got %d", response.Changes[0].EntityID)
	}
}

func TestSyncHandler_DeviceNotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	user := createTestUser(t, db, "syncuser", "password123")

	reqBody := SyncRequest{
		DeviceID:   99999,
		LastSyncAt: nil,
		Changes:    []ChangeLogEntry{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestSyncHandler_InvalidEntityType(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	user := createTestUser(t, db, "syncuser", "password123")
	ctx := context.Background()

	var deviceID int
	err := db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.ID, "mobile", "hash123", time.Now()).Scan(&deviceID)
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	reqBody := SyncRequest{
		DeviceID:   deviceID,
		LastSyncAt: nil,
		Changes: []ChangeLogEntry{
			{
				EntityType: "invalid_type",
				EntityID:   1,
				Operation:  "create",
				OccurredAt: time.Now(),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestSyncHandler_InvalidOperation(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	user := createTestUser(t, db, "syncuser", "password123")
	ctx := context.Background()

	var deviceID int
	err := db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.ID, "mobile", "hash123", time.Now()).Scan(&deviceID)
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	reqBody := SyncRequest{
		DeviceID:   deviceID,
		LastSyncAt: nil,
		Changes: []ChangeLogEntry{
			{
				EntityType: "category",
				EntityID:   1,
				Operation:  "invalid_op",
				OccurredAt: time.Now(),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestSyncHandler_NoAuth(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	reqBody := SyncRequest{
		DeviceID:   1,
		LastSyncAt: nil,
		Changes:    []ChangeLogEntry{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestSyncHandler_WrongMethod(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	user := createTestUser(t, db, "syncuser", "password123")

	req := httptest.NewRequest(http.MethodGet, "/sync", nil)
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestSyncHandler_MultipleChanges(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	api := NewAPI(db, "test-secret")

	user := createTestUser(t, db, "syncuser", "password123")
	ctx := context.Background()

	var deviceID int
	err := db.QueryRow(ctx, `
		INSERT INTO devices (user_id, platform, token_hash, registered_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.ID, "mobile", "hash123", time.Now()).Scan(&deviceID)
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	reqBody := SyncRequest{
		DeviceID:   deviceID,
		LastSyncAt: nil,
		Changes: []ChangeLogEntry{
			{
				EntityType: "category",
				EntityID:   1,
				Operation:  "create",
				OccurredAt: time.Now(),
			},
			{
				EntityType: "template_group",
				EntityID:   2,
				Operation:  "update",
				OccurredAt: time.Now(),
			},
			{
				EntityType: "day_record",
				EntityID:   3,
				Operation:  "delete",
				OccurredAt: time.Now(),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUserID(context.Background(), user.ID))

	w := httptest.NewRecorder()

	// Act
	api.syncHandler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM change_log WHERE device_id = $1", deviceID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query change_log: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 changes recorded, got %d", count)
	}
}
