package main

import (
	"context"
	"testing"
)

func TestUserRepository_DeleteAccount_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()
	user := createTestUser(t, db, "testuser", "password123")

	// Act
	err := repo.DeleteAccount(ctx, user.ID, "password123")

	// Assert
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	// Verify user is deleted
	_, findErr := repo.FindByUsername(ctx, "testuser")
	if findErr == nil {
		t.Error("Expected error when finding deleted user, got nil")
	}
}

func TestUserRepository_DeleteAccount_WrongPassword(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()
	user := createTestUser(t, db, "testuser", "password123")

	// Act
	err := repo.DeleteAccount(ctx, user.ID, "wrongpassword")

	// Assert
	if err != ErrInvalidPassword {
		t.Errorf("Expected ErrInvalidPassword, got %v", err)
	}

	// Verify user still exists
	foundUser, findErr := repo.FindByUsername(ctx, "testuser")
	if findErr != nil {
		t.Errorf("User should still exist after failed delete: %v", findErr)
	}
	if foundUser.ID != user.ID {
		t.Error("Found different user after failed delete")
	}
}

func TestUserRepository_DeleteAccount_NonExistentUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Act
	err := repo.DeleteAccount(ctx, 99999, "anypassword")

	// Assert
	if err == nil {
		t.Error("Expected error when deleting non-existent user, got nil")
	}
}

func TestUserRepository_DeleteAccount_CascadeSettings(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	settingsRepo := NewUserSettingsRepository(db)
	ctx := context.Background()
	user := createTestUser(t, db, "testuser", "password123")

	// Create settings for the user
	_, err := settingsRepo.GetOrCreate(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to create settings: %v", err)
	}

	// Act
	err = userRepo.DeleteAccount(ctx, user.ID, "password123")

	// Assert
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	// Verify settings are cascaded deleted
	var count int
	queryErr := db.QueryRow(ctx, "SELECT COUNT(*) FROM user_settings WHERE user_id = $1", user.ID).Scan(&count)
	if queryErr != nil {
		t.Fatalf("Failed to query settings: %v", queryErr)
	}
	if count != 0 {
		t.Errorf("Expected 0 settings after user deletion, got %d", count)
	}
}

func TestUserRepository_Create_DuplicateUsername(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()
	createTestUser(t, db, "testuser", "password123")

	// Act
	_, err := repo.Create(ctx, "testuser", "differentpassword")

	// Assert
	if err != ErrDuplicateUsername {
		t.Errorf("Expected ErrDuplicateUsername, got %v", err)
	}
}

func TestUserRepository_VerifyPassword(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	user := createTestUser(t, db, "testuser", "password123")

	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"Correct password", "password123", true},
		{"Wrong password", "wrongpassword", false},
		{"Empty password", "", false},
		{"Similar password", "password124", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := repo.VerifyPassword(user, tt.password)

			// Assert
			if result != tt.expected {
				t.Errorf("VerifyPassword(%q) = %v, expected %v", tt.password, result, tt.expected)
			}
		})
	}
}
