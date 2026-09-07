package main

import (
	"context"
	"testing"
)

func TestUserSettingsRepository_Get(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserSettingsRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	settings, err := repo.Get(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if settings.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, settings.UserID)
	}
	if settings.DayBoundaryTime != mustScheduleTime("04:00:00") {
		t.Errorf("Expected default DayBoundaryTime '04:00:00', got '%v'", settings.DayBoundaryTime)
	}
}

func TestUserSettingsRepository_Get_Idempotent(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserSettingsRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	settings1, err1 := repo.Get(ctx, user.ID)
	settings2, err2 := repo.Get(ctx, user.ID)

	// Assert
	if err1 != nil {
		t.Fatalf("First Get failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Second Get failed: %v", err2)
	}
	if settings1.ID != settings2.ID {
		t.Errorf("Expected same settings ID, got %d and %d", settings1.ID, settings2.ID)
	}
}

func TestUserSettingsRepository_Update(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserSettingsRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()
	_, err := repo.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Act
	newTime := mustScheduleTime("06:30:00")
	updated, err := repo.Update(ctx, user.ID, newTime)

	// Assert
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.DayBoundaryTime != newTime {
		t.Errorf("Expected DayBoundaryTime '%v', got '%v'", newTime, updated.DayBoundaryTime)
	}
	if updated.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, updated.UserID)
	}
}

func TestUserSettingsRepository_Update_BeforeCreate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserSettingsRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()
	if _, err := db.Exec(ctx, `DELETE FROM user_settings WHERE user_id = $1`, user.ID); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Act
	_, err := repo.Update(ctx, user.ID, mustScheduleTime("05:00:00"))

	// Assert
	if err == nil {
		t.Error("Expected error when updating non-existent settings, got nil")
	}
}

func TestUserSettingsRepository_MultipleUsers(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserSettingsRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password456")
	ctx := context.Background()

	// Act
	settings1, err := repo.Get(ctx, user1.ID)
	if err != nil {
		t.Fatalf("User1 setup failed: %v", err)
	}
	settings2, err := repo.Get(ctx, user2.ID)
	if err != nil {
		t.Fatalf("User2 setup failed: %v", err)
	}
	if _, err := repo.Update(ctx, user1.ID, mustScheduleTime("05:00:00")); err != nil {
		t.Fatalf("User1 update failed: %v", err)
	}
	if _, err := repo.Update(ctx, user2.ID, mustScheduleTime("07:00:00")); err != nil {
		t.Fatalf("User2 update failed: %v", err)
	}

	updated1, err1 := repo.Get(ctx, user1.ID)
	updated2, err2 := repo.Get(ctx, user2.ID)

	// Assert
	if err1 != nil || err2 != nil {
		t.Fatalf("Get failed: %v, %v", err1, err2)
	}
	if settings1.ID == settings2.ID {
		t.Error("Expected different settings IDs for different users")
	}
	if updated1.DayBoundaryTime != mustScheduleTime("05:00:00") {
		t.Errorf("User1: expected '05:00:00', got '%v'", updated1.DayBoundaryTime)
	}
	if updated2.DayBoundaryTime != mustScheduleTime("07:00:00") {
		t.Errorf("User2: expected '07:00:00', got '%v'", updated2.DayBoundaryTime)
	}
}
