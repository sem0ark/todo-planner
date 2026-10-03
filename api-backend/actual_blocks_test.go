package main

import (
	"errors"
	"testing"
)

func TestComputeTimelineEventConfigurations(t *testing.T) {
	firstCategoryID := 1
	secondCategoryID := 2
	amendedClientEventID := "transition-1"
	amendedCorrectedTime := parseTime("2026-07-20T10:00:00Z")
	testCases := []struct {
		name              string
		events            []DayEvent
		expectedTypes     []string
		expectedDurations []int
	}{
		{name: "empty event list", expectedTypes: []string{}, expectedDurations: []int{}},
		{
			name:          "confirmations only",
			events:        []DayEvent{{ID: 1, EventType: "confirmation", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")}},
			expectedTypes: []string{}, expectedDurations: []int{},
		},
		{
			name: "transition without a later confirmation has no materialized block",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
			},
			expectedTypes: []string{}, expectedDurations: []int{},
		},
		{
			name: "confirmations do not split a block",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "confirmation", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T10:00:00Z")},
				{ID: 3, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
			},
			expectedTypes: []string{"actual"}, expectedDurations: []int{180},
		},
		{
			name: "latest confirmation closes an open block",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
				{ID: 3, EventType: "confirmation", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T14:00:00Z")},
			},
			expectedTypes: []string{"actual", "actual"}, expectedDurations: []int{180, 120},
		},
		{
			name: "confirmation records the current category",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "confirmation", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
			},
			expectedTypes: []string{"actual"}, expectedDurations: []int{180},
		},
		{
			name: "confirmation with a changed category splits the timeline",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "confirmation", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T11:00:00Z")},
				{ID: 3, EventType: "confirmation", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T13:00:00Z")},
			},
			expectedTypes: []string{"actual", "actual"}, expectedDurations: []int{120, 120},
		},
		{
			name: "short intervals are removed",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T09:04:00Z")},
				{ID: 3, EventType: "confirmation", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T09:20:00Z")},
			},
			expectedTypes: []string{"actual"}, expectedDurations: []int{16},
		},
		{
			name: "five minute intervals are retained",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T09:05:00Z")},
			},
			expectedTypes: []string{"actual"}, expectedDurations: []int{5},
		},
		{
			name: "same category intervals merge after a short interruption",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T10:00:00Z")},
				{ID: 3, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T10:02:00Z")},
				{ID: 4, EventType: "confirmation", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T10:20:00Z")},
			},
			expectedTypes: []string{"actual"}, expectedDurations: []int{80},
		},
		{
			name: "amendment corrected timestamp is used",
			events: []DayEvent{
				{ID: 1, EventType: "transition", ClientEventID: &amendedClientEventID, CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "amendment", TargetClientEventID: &amendedClientEventID, CorrectedAt: &amendedCorrectedTime, OccurredAt: parseTime("2026-07-20T10:05:00Z")},
				{ID: 3, EventType: "confirmation", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
			},
			expectedTypes: []string{"actual"}, expectedDurations: []int{120},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			blocks, err := computeTimeline(testCase.events)
			if err != nil {
				t.Fatalf("computeTimeline failed: %v", err)
			}
			if len(blocks) != len(testCase.expectedTypes) {
				t.Fatalf("expected %d blocks, got %d: %+v", len(testCase.expectedTypes), len(blocks), blocks)
			}
			for blockIndex, block := range blocks {
				if block.BlockType != testCase.expectedTypes[blockIndex] {
					t.Errorf("block %d type: expected %q, got %q", blockIndex, testCase.expectedTypes[blockIndex], block.BlockType)
				}
				if block.DurationMinutes != testCase.expectedDurations[blockIndex] {
					t.Errorf("block %d duration: expected %d, got %d", blockIndex, testCase.expectedDurations[blockIndex], block.DurationMinutes)
				}
			}
		})
	}
}

func TestComputeTimelineAmendmentChangesEffectiveTime(t *testing.T) {
	categoryID := 1
	targetEventID := "transition-1"
	correctedTime := parseTime("2026-07-20T11:00:00Z")
	events := []DayEvent{
		{ID: 1, EventType: "transition", ClientEventID: &targetEventID, CategoryID: &categoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
		{ID: 2, EventType: "amendment", TargetClientEventID: &targetEventID, CorrectedAt: &correctedTime, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
		{ID: 3, EventType: "confirmation", CategoryID: &categoryID, OccurredAt: parseTime("2026-07-20T13:00:00Z")},
	}
	blocks, err := computeTimeline(events)
	if err != nil {
		t.Fatalf("computeTimeline failed: %v", err)
	}
	if blocks[0].StartTime != correctedTime {
		t.Fatalf("expected amended start %v, got %v", correctedTime, blocks[0].StartTime)
	}
}

func TestComputeTimelineEqualEffectiveTransitionsReturnConflict(t *testing.T) {
	categoryID := 1
	transitionTime := parseTime("2026-07-20T09:00:00Z")
	events := []DayEvent{
		{ID: 1, EventType: "transition", CategoryID: &categoryID, OccurredAt: transitionTime},
		{ID: 2, EventType: "transition", CategoryID: &categoryID, OccurredAt: transitionTime},
	}
	_, err := computeTimeline(events)
	if !errors.Is(err, ErrNonMonotonicTransitions) {
		t.Fatalf("expected non-monotonic transition error, got %v", err)
	}
}

func TestNormalizeTimelineBlocksRemovesShortBlocksAndMergesCategories(t *testing.T) {
	// Arrange
	categoryID := 7
	blocks := []timelineBlock{
		{categoryID: &categoryID, start: parseTime("2026-07-20T09:00:00Z"), end: parseTime("2026-07-20T10:00:00Z")},
		{categoryID: nil, start: parseTime("2026-07-20T10:00:00Z"), end: parseTime("2026-07-20T10:02:00Z")},
		{categoryID: &categoryID, start: parseTime("2026-07-20T10:02:00Z"), end: parseTime("2026-07-20T10:10:00Z")},
	}

	// Act
	normalizedBlocks := normalizeTimelineBlocks(blocks)

	// Assert
	if len(normalizedBlocks) != 1 {
		t.Fatalf("expected one normalized block, got %d", len(normalizedBlocks))
	}
	if normalizedBlocks[0].start != blocks[0].start {
		t.Errorf("expected merged start %v, got %v", blocks[0].start, normalizedBlocks[0].start)
	}
	if normalizedBlocks[0].end != blocks[2].end {
		t.Errorf("expected merged end %v, got %v", blocks[2].end, normalizedBlocks[0].end)
	}
}

func TestNormalizeTimelineBlocksKeepsFiveMinuteBlock(t *testing.T) {
	// Arrange
	categoryID := 7
	blocks := []timelineBlock{{
		categoryID: &categoryID,
		start:      parseTime("2026-07-20T09:00:00Z"),
		end:        parseTime("2026-07-20T09:05:00Z"),
	}}

	// Act
	normalizedBlocks := normalizeTimelineBlocks(blocks)

	// Assert
	if len(normalizedBlocks) != 1 {
		t.Fatalf("expected boundary-duration block to remain, got %d blocks", len(normalizedBlocks))
	}
}

func TestResolveOverlappingActualBlocksLaterBlockWins(t *testing.T) {
	// Arrange
	firstCategoryID := 1
	secondCategoryID := 2
	blocks := []ActualBlockInput{
		{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("09:00:00"), DurationMinutes: 120},
		{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 30},
	}

	// Act
	resolvedBlocks := resolveOverlappingActualBlocks(blocks)

	// Assert
	if len(resolvedBlocks) != 3 {
		t.Fatalf("expected earlier block to be split around later block, got %d blocks", len(resolvedBlocks))
	}
	if resolvedBlocks[0].StartTime != mustScheduleTime("09:00:00") || resolvedBlocks[0].DurationMinutes != 60 {
		t.Errorf("unexpected left block: %+v", resolvedBlocks[0])
	}
	if resolvedBlocks[1].StartTime != mustScheduleTime("10:00:00") || resolvedBlocks[1].DurationMinutes != 30 {
		t.Errorf("unexpected later block: %+v", resolvedBlocks[1])
	}
	if resolvedBlocks[2].StartTime != mustScheduleTime("10:30:00") || resolvedBlocks[2].DurationMinutes != 30 {
		t.Errorf("unexpected right block: %+v", resolvedBlocks[2])
	}
}

func TestResolveOverlappingActualBlocksRemovesFullyCoveredEarlierBlock(t *testing.T) {
	// Arrange
	firstCategoryID := 1
	secondCategoryID := 2
	blocks := []ActualBlockInput{
		{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("09:30:00"), DurationMinutes: 30},
		{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("09:00:00"), DurationMinutes: 120},
	}

	// Act
	resolvedBlocks := resolveOverlappingActualBlocks(blocks)

	// Assert
	if len(resolvedBlocks) != 1 {
		t.Fatalf("expected fully covered block to be removed, got %d blocks", len(resolvedBlocks))
	}
	if resolvedBlocks[0].CategoryID == nil || *resolvedBlocks[0].CategoryID != secondCategoryID {
		t.Errorf("expected later block category %d, got %v", secondCategoryID, resolvedBlocks[0].CategoryID)
	}
}

func TestResolveOverlappingActualBlocksHandlesOverlapShapes(t *testing.T) {
	// Arrange
	firstCategoryID := 1
	secondCategoryID := 2
	testCases := []struct {
		name     string
		blocks   []ActualBlockInput
		expected []ActualBlockInput
	}{
		{
			name: "later block overlaps the beginning",
			blocks: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 60},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("09:30:00"), DurationMinutes: 45},
			},
			expected: []ActualBlockInput{
				{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("09:30:00"), DurationMinutes: 45},
				{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:15:00"), DurationMinutes: 45},
			},
		},
		{
			name: "later block overlaps the end",
			blocks: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 60},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:45:00"), DurationMinutes: 45},
			},
			expected: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 45},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:45:00"), DurationMinutes: 45},
			},
		},
		{
			name: "later block replaces same start",
			blocks: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 60},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 30},
			},
			expected: []ActualBlockInput{
				{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 30},
				{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:30:00"), DurationMinutes: 30},
			},
		},
		{
			name: "adjacent blocks do not overlap",
			blocks: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 30},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:30:00"), DurationMinutes: 30},
			},
			expected: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:00:00"), DurationMinutes: 30},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartTime: mustScheduleTime("10:30:00"), DurationMinutes: 30},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Act
			resolvedBlocks := resolveOverlappingActualBlocks(testCase.blocks)

			// Assert
			if len(resolvedBlocks) != len(testCase.expected) {
				t.Fatalf("expected %d blocks, got %d", len(testCase.expected), len(resolvedBlocks))
			}
			for blockIndex, expectedBlock := range testCase.expected {
				actualBlock := resolvedBlocks[blockIndex]
				if actualBlock.StartTime != expectedBlock.StartTime || actualBlock.DurationMinutes != expectedBlock.DurationMinutes {
					t.Errorf("block %d: expected %+v, got %+v", blockIndex, expectedBlock, actualBlock)
				}
				if actualBlock.CategoryID == nil || expectedBlock.CategoryID == nil || *actualBlock.CategoryID != *expectedBlock.CategoryID {
					t.Errorf("block %d: expected category %v, got %v", blockIndex, expectedBlock.CategoryID, actualBlock.CategoryID)
				}
			}
		})
	}
}

func TestTrimTimelineBlocksPreservesCrossingCategory(t *testing.T) {
	// Arrange
	categoryID := 7
	checkpoint := parseTime("2026-07-20T10:00:00Z")
	blocks := []timelineBlock{
		{categoryID: &categoryID, start: parseTime("2026-07-20T09:00:00Z"), end: parseTime("2026-07-20T10:15:00Z")},
		{categoryID: &categoryID, start: parseTime("2026-07-20T10:15:00Z"), end: parseTime("2026-07-20T11:00:00Z")},
		{categoryID: &categoryID, start: parseTime("2026-07-20T08:00:00Z"), end: parseTime("2026-07-20T09:00:00Z")},
	}

	// Act
	trimmedBlocks := trimTimelineBlocks(blocks, checkpoint)

	// Assert
	if len(trimmedBlocks) != 2 {
		t.Fatalf("expected two blocks after checkpoint, got %d", len(trimmedBlocks))
	}
	if trimmedBlocks[0].start != checkpoint || trimmedBlocks[0].end != blocks[0].end {
		t.Errorf("expected crossing block to start at checkpoint, got %+v", trimmedBlocks[0])
	}
}

func TestComputeIncrementalTimelineUsesSavedCheckpoint(t *testing.T) {
	// Arrange
	categoryID := 1
	newCategoryID := 2
	existingBlocks := []ActualBlock{{
		BlockType:       "actual",
		CategoryID:      &categoryID,
		StartTime:       mustScheduleTime("09:00:00"),
		DurationMinutes: 60,
	}}
	events := []DayEvent{
		{ID: 1, EventType: "transition", CategoryID: &newCategoryID, OccurredAt: parseTime("2026-07-20T09:59:00Z")},
		{ID: 2, EventType: "confirmation", CategoryID: &newCategoryID, OccurredAt: parseTime("2026-07-20T10:00:00Z")},
		{ID: 3, EventType: "transition", CategoryID: &newCategoryID, OccurredAt: parseTime("2026-07-20T10:10:00Z")},
	}

	// Act
	computedBlocks, err := computeIncrementalTimeline(events, existingBlocks, mustCalendarDate("2026-07-20"))

	// Assert
	if err != nil {
		t.Fatalf("computeIncrementalTimeline failed: %v", err)
	}
	if len(computedBlocks) != 1 {
		t.Fatalf("expected one appended block, got %d", len(computedBlocks))
	}
	if computedBlocks[0].StartTime != parseTime("2026-07-20T10:00:00Z") {
		t.Errorf("expected checkpoint event to be ignored, got start %v", computedBlocks[0].StartTime)
	}
	if computedBlocks[0].DurationMinutes != 10 {
		t.Errorf("expected ten-minute block, got %d minutes", computedBlocks[0].DurationMinutes)
	}
}

func TestComputeIncrementalTimelineTrimsAmendedHistoricalEvent(t *testing.T) {
	// Arrange
	categoryID := 1
	newCategoryID := 2
	clientEventID := "old-transition"
	targetClientEventID := "old-transition"
	correctedTime := parseTime("2026-07-20T10:30:00Z")
	existingBlocks := []ActualBlock{{
		BlockType:       "actual",
		CategoryID:      &categoryID,
		StartTime:       mustScheduleTime("09:00:00"),
		DurationMinutes: 60,
	}}
	events := []DayEvent{
		{ID: 1, EventType: "amendment", TargetClientEventID: &targetClientEventID, CorrectedAt: &correctedTime, OccurredAt: parseTime("2026-07-20T11:00:00Z")},
		{ID: 2, EventType: "transition", ClientEventID: &clientEventID, CategoryID: &newCategoryID, OccurredAt: parseTime("2026-07-20T09:30:00Z")},
		{ID: 3, EventType: "confirmation", CategoryID: &categoryID, OccurredAt: parseTime("2026-07-20T11:00:00Z")},
		{ID: 4, EventType: "transition", CategoryID: &categoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
	}

	// Act
	computedBlocks, err := computeIncrementalTimeline(events, existingBlocks, mustCalendarDate("2026-07-20"))

	// Assert
	if err != nil {
		t.Fatalf("computeIncrementalTimeline failed: %v", err)
	}
	if len(computedBlocks) != 2 {
		t.Fatalf("expected two blocks after trimming, got %+v", computedBlocks)
	}
	if computedBlocks[0].StartTime != parseTime("2026-07-20T10:00:00Z") || computedBlocks[0].DurationMinutes != 30 {
		t.Errorf("expected trimmed block from 10:00 to 10:30, got %+v", computedBlocks[0])
	}
}
