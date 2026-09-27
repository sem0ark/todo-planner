package main

import "testing"

func TestAdjustActualBlocksForStorage(t *testing.T) {
	categoryID := 21
	blocks := []ActualBlockInput{{
		CategoryID:      &categoryID,
		BlockType:       "actual",
		StartTime:       mustScheduleTime("21:50:00"),
		DurationMinutes: 135,
	}}

	adjustedBlocks := adjustActualBlocksForStorage(blocks, 120)

	if adjustedBlocks[0].StartTime != mustScheduleTime("19:50:00") {
		t.Fatalf("expected local block to be stored at 19:50:00, got %v", adjustedBlocks[0].StartTime)
	}
	if blocks[0].StartTime != mustScheduleTime("21:50:00") {
		t.Fatalf("expected input blocks to remain unchanged, got %v", blocks[0].StartTime)
	}
}
