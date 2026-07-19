package main

import (
	"context"
	"testing"
)

func TestDeviceRepository_Create(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	device, err := repo.Create(ctx, user.ID, "mobile")

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if device.ID == 0 {
		t.Error("Expected device ID to be set")
	}
	if device.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, device.UserID)
	}
	if device.Platform != "mobile" {
		t.Errorf("Expected platform 'mobile', got '%s'", device.Platform)
	}
	if device.RegisteredAt.IsZero() {
		t.Error("Expected RegisteredAt to be set")
	}
}

func TestDeviceRepository_Create_DifferentPlatforms(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	platforms := []string{"desktop", "mobile", "web"}

	for _, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			// Act
			device, err := repo.Create(ctx, user.ID, platform)

			// Assert
			if err != nil {
				t.Fatalf("Create failed for platform %s: %v", platform, err)
			}
			if device.Platform != platform {
				t.Errorf("Expected platform '%s', got '%s'", platform, device.Platform)
			}
		})
	}
}

func TestDeviceRepository_Create_MultipleDevices(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act - create multiple devices for same user
	device1, err1 := repo.Create(ctx, user.ID, "desktop")
	device2, err2 := repo.Create(ctx, user.ID, "mobile")

	// Assert
	if err1 != nil || err2 != nil {
		t.Fatalf("Create failed: %v, %v", err1, err2)
	}
	if device1.ID == device2.ID {
		t.Error("Expected different device IDs")
	}
	if device1.UserID != user.ID || device2.UserID != user.ID {
		t.Error("Both devices should belong to the same user")
	}
}
