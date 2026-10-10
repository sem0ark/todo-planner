package main

import (
	"context"
	"testing"
)

func TestActualBlockRepository_ReplaceAndFindByRecordID(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "actual-block-repository", "password123")
	record, err := newDayTrackingServiceForTest(database).CreateDayRecord(
		context.Background(), user.ID, mustCalendarDate("2026-09-06"),
	)
	if err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	blockRepository := NewActualBlockRepository(database)
	transaction, err := database.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	createdBlocks, err := blockRepository.ReplaceTx(
		context.Background(),
		transaction,
		record.ID,
		[]ActualBlockInput{{
			BlockType:       "blank",
			StartTime:       mustScheduleTime("09:00:00"),
			DurationMinutes: 30,
		}},
	)
	if err != nil {
		transaction.Rollback(context.Background())
		t.Fatalf("failed to replace blocks: %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("failed to commit blocks: %v", err)
	}
	if len(createdBlocks) != 1 {
		t.Fatalf("expected one created block, got %d", len(createdBlocks))
	}
	loadedBlocks, err := blockRepository.FindByRecordID(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("failed to load blocks: %v", err)
	}
	if len(loadedBlocks) != 1 || loadedBlocks[0].StartTime != mustScheduleTime("09:00:00") {
		t.Fatalf("unexpected loaded blocks: %+v", loadedBlocks)
	}
}
