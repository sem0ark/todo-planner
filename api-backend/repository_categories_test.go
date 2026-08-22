package main

import (
	"context"
	"testing"
)

func TestCategoryRepository_Create(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	input := CategoryInput{
		Name:  "Work",
		Color: "#FF5733",
	}

	// Act
	category, err := repo.Create(ctx, input, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if category.ID == 0 {
		t.Error("Expected category ID to be set")
	}
	if category.Name != input.Name {
		t.Errorf("Expected name '%s', got '%s'", input.Name, category.Name)
	}
	if category.Color != input.Color {
		t.Errorf("Expected color '%s', got '%s'", input.Color, category.Color)
	}
	if category.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, category.UserID)
	}
}

func TestCategoryRepository_CreateWithPomodoroConfig(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUser(t, db, "pomodoro_create_user", "password123")
	ctx := context.Background()
	config := &PomodoroConfig{WorkDuration: 1500, RestDuration: 300}

	// Act
	category, err := repo.Create(ctx, CategoryInput{
		Name:           "Focus",
		Color:          "#FF5733",
		PomodoroConfig: config,
	}, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if category.PomodoroConfig == nil {
		t.Fatal("Expected pomodoro config to be set")
	}
	if *category.PomodoroConfig != *config {
		t.Errorf("Expected pomodoro config %+v, got %+v", config, category.PomodoroConfig)
	}

	fetched, err := repo.FindByID(ctx, category.ID, user.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if fetched.PomodoroConfig == nil || *fetched.PomodoroConfig != *config {
		t.Errorf("Expected persisted pomodoro config %+v, got %+v", config, fetched.PomodoroConfig)
	}
}

func TestCategoryRepository_FindByUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	repo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	repo.Create(ctx, CategoryInput{Name: "Personal", Color: "#33FF57"}, user.ID)

	// Act
	categories, err := repo.FindByUser(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByUser failed: %v", err)
	}
	if len(categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(categories))
	}
}

func TestCategoryRepository_FindByUser_ExcludesDeleted(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	cat1, _ := repo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	repo.Create(ctx, CategoryInput{Name: "Personal", Color: "#33FF57"}, user.ID)
	repo.Delete(ctx, cat1.ID, user.ID)

	// Act
	categories, err := repo.FindByUser(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByUser failed: %v", err)
	}
	if len(categories) != 1 {
		t.Errorf("Expected 1 active category, got %d", len(categories))
	}
	if categories[0].Name != "Personal" {
		t.Errorf("Expected remaining category to be 'Personal', got '%s'", categories[0].Name)
	}
}

func TestCategoryRepository_FindByUser_MultipleUsers(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	repo.Create(ctx, CategoryInput{Name: "User1 Work", Color: "#FF5733"}, user1.ID)
	repo.Create(ctx, CategoryInput{Name: "User2 Work", Color: "#33FF57"}, user2.ID)

	// Act
	categories1, _ := repo.FindByUser(ctx, user1.ID)
	categories2, _ := repo.FindByUser(ctx, user2.ID)

	// Assert
	if len(categories1) != 1 || len(categories2) != 1 {
		t.Error("Each user should only see their own categories")
	}
	if categories1[0].Name != "User1 Work" {
		t.Errorf("User1 should see 'User1 Work', got '%s'", categories1[0].Name)
	}
	if categories2[0].Name != "User2 Work" {
		t.Errorf("User2 should see 'User2 Work', got '%s'", categories2[0].Name)
	}
}

func TestCategoryRepository_Update(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	category, _ := repo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	newInput := CategoryInput{
		Name:  "Deep Work",
		Color: "#0000FF",
	}

	// Act
	updated, err := repo.Update(ctx, category.ID, newInput, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != newInput.Name {
		t.Errorf("Expected name '%s', got '%s'", newInput.Name, updated.Name)
	}
	if updated.Color != newInput.Color {
		t.Errorf("Expected color '%s', got '%s'", newInput.Color, updated.Color)
	}
	if updated.ID != category.ID {
		t.Error("Category ID should not change")
	}
}

func TestCategoryRepository_UpdateClearsPomodoroConfig(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUser(t, db, "pomodoro_update_user", "password123")
	ctx := context.Background()
	category, err := repo.Create(ctx, CategoryInput{
		Name:           "Focus",
		Color:          "#FF5733",
		PomodoroConfig: &PomodoroConfig{WorkDuration: 1500, RestDuration: 300},
	}, user.ID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Act
	updated, err := repo.Update(ctx, category.ID, CategoryInput{
		Name:  "Focus",
		Color: "#FF5733",
	}, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.PomodoroConfig != nil {
		t.Errorf("Expected pomodoro config to be cleared, got %+v", updated.PomodoroConfig)
	}
}

func TestCategoryRepository_Update_WrongUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	category, _ := repo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user1.ID)

	// Act
	_, err := repo.Update(ctx, category.ID, CategoryInput{Name: "Hacked", Color: "#000000"}, user2.ID)

	// Assert
	if err == nil {
		t.Error("Expected error when updating another user's category")
	}
}

func TestCategoryRepository_Delete(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	category, _ := repo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	// Act
	err := repo.Delete(ctx, category.ID, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	categories, _ := repo.FindByUser(ctx, user.ID)
	if len(categories) != 0 {
		t.Error("Deleted category should not appear in FindByUser")
	}

	deletedCat, _ := repo.FindByID(ctx, category.ID, user.ID)
	if !deletedCat.IsDeleted {
		t.Error("Category should be marked as deleted")
	}
}

func TestCategoryRepository_Delete_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	err := repo.Delete(ctx, 99999, user.ID)

	// Assert
	if err != ErrCategoryNotFound {
		t.Errorf("Expected ErrCategoryNotFound, got %v", err)
	}
}

func TestCategoryRepository_Delete_WrongUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	category, _ := repo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user1.ID)

	// Act
	err := repo.Delete(ctx, category.ID, user2.ID)

	// Assert
	if err != ErrCategoryNotFound {
		t.Errorf("Expected ErrCategoryNotFound, got %v", err)
	}

	categories, _ := repo.FindByUser(ctx, user1.ID)
	if len(categories) != 1 {
		t.Error("User1's category should still exist")
	}
}
