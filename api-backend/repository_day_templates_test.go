package main

import (
	"context"
	"testing"
)

func TestDayTemplateRepository_Create(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	categoryRepo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	category, _ := categoryRepo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	input := DayTemplateInput{
		Name: "Weekday",
		PlannedBlocks: []PlannedBlockInput{
			{CategoryID: category.ID, StartTime: "09:00:00", DurationMinutes: 120},
			{CategoryID: category.ID, StartTime: "13:00:00", DurationMinutes: 180},
		},
	}

	// Act
	template, err := repo.Create(ctx, input, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if template.ID == 0 {
		t.Error("Expected template ID to be set")
	}
	if template.Name != input.Name {
		t.Errorf("Expected name '%s', got '%s'", input.Name, template.Name)
	}
	if len(template.PlannedBlocks) != 2 {
		t.Errorf("Expected 2 planned blocks, got %d", len(template.PlannedBlocks))
	}
}

func TestDayTemplateRepository_Create_WithTemplateGroup(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	templateRepo := NewDayTemplateRepository(db)
	groupRepo := NewTemplateGroupRepository(db)
	categoryRepo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	group, _ := groupRepo.Create(ctx, TemplateGroupInput{Name: "Work"}, user.ID)
	category, _ := categoryRepo.Create(ctx, CategoryInput{Name: "Deep Work", Color: "#FF5733"}, user.ID)

	input := DayTemplateInput{
		Name:            "Weekday",
		TemplateGroupID: &group.ID,
		PlannedBlocks: []PlannedBlockInput{
			{CategoryID: category.ID, StartTime: "09:00:00", DurationMinutes: 480},
		},
	}

	// Act
	template, err := templateRepo.Create(ctx, input, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if template.TemplateGroupID == nil || *template.TemplateGroupID != group.ID {
		t.Errorf("Expected TemplateGroupID %d, got %v", group.ID, template.TemplateGroupID)
	}
}

func TestDayTemplateRepository_FindByUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	repo.Create(ctx, DayTemplateInput{Name: "Weekday", PlannedBlocks: []PlannedBlockInput{}}, user.ID)
	repo.Create(ctx, DayTemplateInput{Name: "Weekend", PlannedBlocks: []PlannedBlockInput{}}, user.ID)

	// Act
	templates, err := repo.FindByUser(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByUser failed: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(templates))
	}
}

func TestDayTemplateRepository_FindByUser_ExcludesDeleted(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	template1, _ := repo.Create(ctx, DayTemplateInput{Name: "Weekday", PlannedBlocks: []PlannedBlockInput{}}, user.ID)
	repo.Create(ctx, DayTemplateInput{Name: "Weekend", PlannedBlocks: []PlannedBlockInput{}}, user.ID)
	repo.Delete(ctx, template1.ID, user.ID)

	// Act
	templates, err := repo.FindByUser(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByUser failed: %v", err)
	}
	if len(templates) != 1 {
		t.Errorf("Expected 1 active template, got %d", len(templates))
	}
	if templates[0].Name != "Weekend" {
		t.Errorf("Expected remaining template to be 'Weekend', got '%s'", templates[0].Name)
	}
}

func TestDayTemplateRepository_FindByUser_IncludesBlocks(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	templateRepo := NewDayTemplateRepository(db)
	categoryRepo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	category, _ := categoryRepo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)

	input := DayTemplateInput{
		Name: "Weekday",
		PlannedBlocks: []PlannedBlockInput{
			{CategoryID: category.ID, StartTime: "09:00:00", DurationMinutes: 480},
		},
	}
	templateRepo.Create(ctx, input, user.ID)

	// Act
	templates, err := templateRepo.FindByUser(ctx, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("FindByUser failed: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template, got %d", len(templates))
	}
	if len(templates[0].PlannedBlocks) != 1 {
		t.Errorf("Expected 1 planned block, got %d", len(templates[0].PlannedBlocks))
	}
}

func TestDayTemplateRepository_FindByUser_MultipleUsers(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	repo.Create(ctx, DayTemplateInput{Name: "User1 Template", PlannedBlocks: []PlannedBlockInput{}}, user1.ID)
	repo.Create(ctx, DayTemplateInput{Name: "User2 Template", PlannedBlocks: []PlannedBlockInput{}}, user2.ID)

	// Act
	templates1, _ := repo.FindByUser(ctx, user1.ID)
	templates2, _ := repo.FindByUser(ctx, user2.ID)

	// Assert
	if len(templates1) != 1 || len(templates2) != 1 {
		t.Error("Each user should only see their own templates")
	}
	if templates1[0].Name != "User1 Template" {
		t.Errorf("User1 should see 'User1 Template', got '%s'", templates1[0].Name)
	}
	if templates2[0].Name != "User2 Template" {
		t.Errorf("User2 should see 'User2 Template', got '%s'", templates2[0].Name)
	}
}

func TestDayTemplateRepository_Update(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	templateRepo := NewDayTemplateRepository(db)
	categoryRepo := NewCategoryRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	category1, _ := categoryRepo.Create(ctx, CategoryInput{Name: "Work", Color: "#FF5733"}, user.ID)
	category2, _ := categoryRepo.Create(ctx, CategoryInput{Name: "Rest", Color: "#33FF57"}, user.ID)

	template, _ := templateRepo.Create(ctx, DayTemplateInput{
		Name: "Original",
		PlannedBlocks: []PlannedBlockInput{
			{CategoryID: category1.ID, StartTime: "09:00:00", DurationMinutes: 480},
		},
	}, user.ID)

	newInput := DayTemplateInput{
		Name: "Updated",
		PlannedBlocks: []PlannedBlockInput{
			{CategoryID: category2.ID, StartTime: "10:00:00", DurationMinutes: 240},
			{CategoryID: category1.ID, StartTime: "14:00:00", DurationMinutes: 120},
		},
	}

	// Act
	updated, err := templateRepo.Update(ctx, template.ID, newInput, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != newInput.Name {
		t.Errorf("Expected name '%s', got '%s'", newInput.Name, updated.Name)
	}
	if len(updated.PlannedBlocks) != 2 {
		t.Errorf("Expected 2 planned blocks, got %d", len(updated.PlannedBlocks))
	}
	if updated.ID != template.ID {
		t.Error("Template ID should not change")
	}
}

func TestDayTemplateRepository_Update_WrongUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	template, _ := repo.Create(ctx, DayTemplateInput{Name: "Original", PlannedBlocks: []PlannedBlockInput{}}, user1.ID)

	// Act
	_, err := repo.Update(ctx, template.ID, DayTemplateInput{Name: "Hacked", PlannedBlocks: []PlannedBlockInput{}}, user2.ID)

	// Assert
	if err == nil {
		t.Error("Expected error when updating another user's template")
	}
}

func TestDayTemplateRepository_Delete(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	template, _ := repo.Create(ctx, DayTemplateInput{Name: "Weekday", PlannedBlocks: []PlannedBlockInput{}}, user.ID)

	// Act
	err := repo.Delete(ctx, template.ID, user.ID)

	// Assert
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	templates, _ := repo.FindByUser(ctx, user.ID)
	if len(templates) != 0 {
		t.Error("Deleted template should not appear in FindByUser")
	}

	deletedTemplate, _ := repo.FindByID(ctx, template.ID, user.ID)
	if !deletedTemplate.IsDeleted {
		t.Error("Template should be marked as deleted")
	}
}

func TestDayTemplateRepository_Delete_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user := createTestUser(t, db, "testuser", "password123")
	ctx := context.Background()

	// Act
	err := repo.Delete(ctx, 99999, user.ID)

	// Assert
	if err != ErrDayTemplateNotFound {
		t.Errorf("Expected ErrDayTemplateNotFound, got %v", err)
	}
}

func TestDayTemplateRepository_Delete_WrongUser(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewDayTemplateRepository(db)
	user1 := createTestUser(t, db, "user1", "password123")
	user2 := createTestUser(t, db, "user2", "password123")
	ctx := context.Background()

	template, _ := repo.Create(ctx, DayTemplateInput{Name: "Weekday", PlannedBlocks: []PlannedBlockInput{}}, user1.ID)

	// Act
	err := repo.Delete(ctx, template.ID, user2.ID)

	// Assert
	if err != ErrDayTemplateNotFound {
		t.Errorf("Expected ErrDayTemplateNotFound, got %v", err)
	}

	templates, _ := repo.FindByUser(ctx, user1.ID)
	if len(templates) != 1 {
		t.Error("User1's template should still exist")
	}
}
