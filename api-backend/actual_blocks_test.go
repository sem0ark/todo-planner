package main

import "testing"

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
			name: "notes behave like confirmations",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "note", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T10:00:00Z")},
				{ID: 3, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
			},
			expectedTypes: []string{"actual"}, expectedDurations: []int{180},
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
			blocks, err := computeDayTimeline(testCase.events)
			if err != nil {
				t.Fatalf("computeDayTimeline failed: %v", err)
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
	blocks, err := computeDayTimeline(events)
	if err != nil {
		t.Fatalf("computeDayTimeline failed: %v", err)
	}
	if !blocks[0].StartAt.Equal(correctedTime) {
		t.Fatalf("expected amended start %v, got %v", correctedTime, blocks[0].StartAt)
	}
}

func TestComputeTimelineEqualEffectiveTransitionsKeepLastObservation(t *testing.T) {
	firstCategoryID := 1
	secondCategoryID := 2
	transitionTime := parseTime("2026-07-20T09:00:00Z")
	endTime := parseTime("2026-07-20T10:00:00Z")
	events := []DayEvent{
		{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: transitionTime},
		{ID: 2, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: transitionTime},
		{ID: 3, EventType: "confirmation", CategoryID: &secondCategoryID, OccurredAt: endTime},
	}

	blocks, err := computeDayTimeline(events)
	if err != nil {
		t.Fatalf("computeDayTimeline failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected one block, got %d: %+v", len(blocks), blocks)
	}
	if blocks[0].CategoryID == nil || *blocks[0].CategoryID != secondCategoryID {
		t.Fatalf("expected last same-time observation category %d, got %+v", secondCategoryID, blocks[0].CategoryID)
	}
	if blocks[0].DurationMinutes != 60 {
		t.Fatalf("expected 60-minute block, got %d", blocks[0].DurationMinutes)
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
		{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T09:00:00Z"), DurationMinutes: 120},
		{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 30},
	}

	// Act
	resolvedBlocks := resolveOverlappingActualBlocks(blocks)

	// Assert
	if len(resolvedBlocks) != 3 {
		t.Fatalf("expected earlier block to be split around later block, got %d blocks", len(resolvedBlocks))
	}
	if !resolvedBlocks[0].StartAt.Equal(parseTime("2026-07-20T09:00:00Z")) || resolvedBlocks[0].DurationMinutes != 60 {
		t.Errorf("unexpected left block: %+v", resolvedBlocks[0])
	}
	if !resolvedBlocks[1].StartAt.Equal(parseTime("2026-07-20T10:00:00Z")) || resolvedBlocks[1].DurationMinutes != 30 {
		t.Errorf("unexpected later block: %+v", resolvedBlocks[1])
	}
	if !resolvedBlocks[2].StartAt.Equal(parseTime("2026-07-20T10:30:00Z")) || resolvedBlocks[2].DurationMinutes != 30 {
		t.Errorf("unexpected right block: %+v", resolvedBlocks[2])
	}
}

func TestResolveOverlappingActualBlocksRemovesFullyCoveredEarlierBlock(t *testing.T) {
	// Arrange
	firstCategoryID := 1
	secondCategoryID := 2
	blocks := []ActualBlockInput{
		{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T09:30:00Z"), DurationMinutes: 30},
		{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T09:00:00Z"), DurationMinutes: 120},
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
				{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 60},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T09:30:00Z"), DurationMinutes: 45},
			},
			expected: []ActualBlockInput{
				{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T09:30:00Z"), DurationMinutes: 45},
				{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:15:00Z"), DurationMinutes: 45},
			},
		},
		{
			name: "later block overlaps the end",
			blocks: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 60},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:45:00Z"), DurationMinutes: 45},
			},
			expected: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 45},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:45:00Z"), DurationMinutes: 45},
			},
		},
		{
			name: "later block replaces same start",
			blocks: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 60},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 30},
			},
			expected: []ActualBlockInput{
				{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 30},
				{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:30:00Z"), DurationMinutes: 30},
			},
		},
		{
			name: "adjacent blocks do not overlap",
			blocks: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 30},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:30:00Z"), DurationMinutes: 30},
			},
			expected: []ActualBlockInput{
				{CategoryID: &firstCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:00:00Z"), DurationMinutes: 30},
				{CategoryID: &secondCategoryID, BlockType: "actual", StartAt: parseTime("2026-07-20T10:30:00Z"), DurationMinutes: 30},
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
				if !actualBlock.StartAt.Equal(expectedBlock.StartAt) || actualBlock.DurationMinutes != expectedBlock.DurationMinutes {
					t.Errorf("block %d: expected %+v, got %+v", blockIndex, expectedBlock, actualBlock)
				}
				if actualBlock.CategoryID == nil || expectedBlock.CategoryID == nil || *actualBlock.CategoryID != *expectedBlock.CategoryID {
					t.Errorf("block %d: expected category %v, got %v", blockIndex, expectedBlock.CategoryID, actualBlock.CategoryID)
				}
			}
		})
	}
}
