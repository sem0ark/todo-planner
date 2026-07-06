package main

import (
	"context"
	"testing"
)

func TestUserSettingsRepository_GetOrCreate(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserSettingsRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	settings, err := repo.GetOrCreate(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if settings.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, settings.UserID)
	}
	if settings.DayBoundaryTime != "04:00:00" {
		t.Errorf("Expected default DayBoundaryTime '04:00:00', got '%s'", settings.DayBoundaryTime)
	}
}

func TestUserSettingsRepository_GetOrCreate_Idempotent(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserSettingsRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	settings1, err1 := repo.GetOrCreate(ctx, user.ID)
	settings2, err2 := repo.GetOrCreate(ctx, user.ID)

	// Assert
	if err1 != nil {
		t.Fatalf("First GetOrCreate failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Second GetOrCreate failed: %v", err2)
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
	_, err := repo.GetOrCreate(ctx, user.ID)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Act
	newTime := "06:30:00"
	updated, err := repo.Update(ctx, user.ID, newTime)

	// Assert
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.DayBoundaryTime != newTime {
		t.Errorf("Expected DayBoundaryTime '%s', got '%s'", newTime, updated.DayBoundaryTime)
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

	// Act
	_, err := repo.Update(ctx, user.ID, "05:00:00")

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
	settings1, _ := repo.GetOrCreate(ctx, user1.ID)
	settings2, _ := repo.GetOrCreate(ctx, user2.ID)
	repo.Update(ctx, user1.ID, "05:00:00")
	repo.Update(ctx, user2.ID, "07:00:00")

	updated1, err1 := repo.GetOrCreate(ctx, user1.ID)
	updated2, err2 := repo.GetOrCreate(ctx, user2.ID)

	// Assert
	if err1 != nil || err2 != nil {
		t.Fatalf("GetOrCreate failed: %v, %v", err1, err2)
	}
	if settings1.ID == settings2.ID {
		t.Error("Expected different settings IDs for different users")
	}
	if updated1.DayBoundaryTime != "05:00:00" {
		t.Errorf("User1: expected '05:00:00', got '%s'", updated1.DayBoundaryTime)
	}
	if updated2.DayBoundaryTime != "07:00:00" {
		t.Errorf("User2: expected '07:00:00', got '%s'", updated2.DayBoundaryTime)
	}
}
