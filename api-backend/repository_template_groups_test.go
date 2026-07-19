package main

import (
	"context"
	"testing"
)

func TestTemplateGroupRepository_Create(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	input := TemplateGroupInput{
		Name: "Full-Time Work",
	}

	// Act
	group, err := repo.Create(ctx, input, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if group.ID == 0 {
		t.Error("Expected group ID to be set")
	}
	if group.Name != input.Name {
		t.Errorf("Expected name '%s', got '%s'", input.Name, group.Name)
	}
	if group.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, group.UserID)
	}
}

func TestTemplateGroupRepository_FindByUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	repo.Create(ctx, TemplateGroupInput{Name: "Work"}, user.ID)
	repo.Create(ctx, TemplateGroupInput{Name: "Vacation"}, user.ID)

	// Act
	groups, err := repo.FindByUser(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByUser failed: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestTemplateGroupRepository_FindByUser_ExcludesDeleted(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	group1, _ := repo.Create(ctx, TemplateGroupInput{Name: "Work"}, user.ID)
	repo.Create(ctx, TemplateGroupInput{Name: "Vacation"}, user.ID)
	repo.Delete(ctx, group1.ID, user.ID)

	// Act
	groups, err := repo.FindByUser(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByUser failed: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("Expected 1 active group, got %d", len(groups))
	}
	if groups[0].Name != "Vacation" {
		t.Errorf("Expected remaining group to be 'Vacation', got '%s'", groups[0].Name)
	}
}

func TestTemplateGroupRepository_FindByUser_MultipleUsers(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	repo.Create(ctx, TemplateGroupInput{Name: "User1 Work"}, user1.ID)
	repo.Create(ctx, TemplateGroupInput{Name: "User2 Work"}, user2.ID)

	// Act
	groups1, _ := repo.FindByUser(ctx, user1.ID)
	groups2, _ := repo.FindByUser(ctx, user2.ID)

	// Assert
	if len(groups1) != 1 || len(groups2) != 1 {
		t.Error("Each user should only see their own groups")
	}
	if groups1[0].Name != "User1 Work" {
		t.Errorf("User1 should see 'User1 Work', got '%s'", groups1[0].Name)
	}
	if groups2[0].Name != "User2 Work" {
		t.Errorf("User2 should see 'User2 Work', got '%s'", groups2[0].Name)
	}
}

func TestTemplateGroupRepository_Update(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	group, _ := repo.Create(ctx, TemplateGroupInput{Name: "Work"}, user.ID)

	newInput := TemplateGroupInput{
		Name: "Full-Time Work",
	}

	// Act
	updated, err := repo.Update(ctx, group.ID, newInput, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != newInput.Name {
		t.Errorf("Expected name '%s', got '%s'", newInput.Name, updated.Name)
	}
	if updated.ID != group.ID {
		t.Error("Group ID should not change")
	}
}

func TestTemplateGroupRepository_Update_WrongUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	group, _ := repo.Create(ctx, TemplateGroupInput{Name: "Work"}, user1.ID)

	// Act
	_, err := repo.Update(ctx, group.ID, TemplateGroupInput{Name: "Hacked"}, user2.ID)

	// Assert
	if err == nil {
		t.Error("Expected error when updating another user's group")
	}
}

func TestTemplateGroupRepository_Delete(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	group, _ := repo.Create(ctx, TemplateGroupInput{Name: "Work"}, user.ID)

	// Act
	err := repo.Delete(ctx, group.ID, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	groups, _ := repo.FindByUser(ctx, user.ID)
	if len(groups) != 0 {
		t.Error("Deleted group should not appear in FindByUser")
	}

	deletedGroup, _ := repo.FindByID(ctx, group.ID, user.ID)
	if !deletedGroup.IsDeleted {
		t.Error("Group should be marked as deleted")
	}
}

func TestTemplateGroupRepository_Delete_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	err := repo.Delete(ctx, 99999, user.ID)

	// Assert
	if err != ErrTemplateGroupNotFound {
		t.Errorf("Expected ErrTemplateGroupNotFound, got %v", err)
	}
}

func TestTemplateGroupRepository_Delete_WrongUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewTemplateGroupRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	group, _ := repo.Create(ctx, TemplateGroupInput{Name: "Work"}, user1.ID)

	// Act
	err := repo.Delete(ctx, group.ID, user2.ID)

	// Assert
	if err != ErrTemplateGroupNotFound {
		t.Errorf("Expected ErrTemplateGroupNotFound, got %v", err)
	}

	groups, _ := repo.FindByUser(ctx, user1.ID)
	if len(groups) != 1 {
		t.Error("User1's group should still exist")
	}
}
