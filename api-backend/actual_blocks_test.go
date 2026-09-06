package main

import (
	"errors"
	"testing"
	"time"
)

func TestComputeTimeline_EventConfigurations(t *testing.T) {
	boundaryStart := parseTime("2026-07-20T04:00:00Z")
	boundaryEnd := boundaryStart.Add(24 * time.Hour)
	now := parseTime("2026-07-20T17:00:00Z")
	firstCategoryID := 1
	secondCategoryID := 2

	testCases := []struct {
		name              string
		events            []DayEvent
		isPastDay         bool
		expectedTypes     []string
		expectedDurations []int
		expectedOpen      []bool
	}{
		{
			name:              "empty event list",
			expectedTypes:     []string{"untracked"},
			expectedDurations: []int{1440},
		},
		{
			name:          "confirmations only",
			events:        []DayEvent{{ID: 1, EventType: "confirmation", OccurredAt: parseTime("2026-07-20T09:00:00Z")}},
			expectedTypes: []string{"untracked"}, expectedDurations: []int{1440},
		},
		{
			name: "confirmation before first transition is orphaned",
			events: []DayEvent{
				{ID: 1, EventType: "confirmation", OccurredAt: parseTime("2026-07-20T08:00:00Z")},
				{ID: 2, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
			},
			expectedTypes: []string{"untracked", "actual"}, expectedDurations: []int{300, 480}, expectedOpen: []bool{false, true},
		},
		{
			name: "confirmations do not split a block",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "confirmation", OccurredAt: parseTime("2026-07-20T10:00:00Z")},
				{ID: 3, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
			},
			expectedTypes: []string{"untracked", "actual", "actual"}, expectedDurations: []int{300, 180, 300}, expectedOpen: []bool{false, false, true},
		},
		{
			name:      "past day closes final block",
			events:    []DayEvent{{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")}},
			isPastDay: true, expectedTypes: []string{"untracked", "actual"}, expectedDurations: []int{300, 1140}, expectedOpen: []bool{false, false},
		},
		{
			name: "out of window events are excluded",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-19T09:00:00Z")},
				{ID: 2, EventType: "confirmation", OccurredAt: parseTime("2026-07-21T09:00:00Z")},
			},
			expectedTypes: []string{"untracked"}, expectedDurations: []int{1440},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			blocks, err := computeTimeline(testCase.events, boundaryStart, boundaryEnd, now, testCase.isPastDay)
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
				if len(testCase.expectedOpen) > blockIndex && block.IsOpen != testCase.expectedOpen[blockIndex] {
					t.Errorf("block %d open state: expected %t, got %t", blockIndex, testCase.expectedOpen[blockIndex], block.IsOpen)
				}
			}
		})
	}
}

func TestComputeTimeline_AmendmentChangesEffectiveTime(t *testing.T) {
	// Arrange
	categoryID := 1
	targetEventID := "transition-1"
	correctedTime := parseTime("2026-07-20T11:00:00Z")
	events := []DayEvent{
		{ID: 1, EventType: "transition", ClientEventID: &targetEventID, CategoryID: &categoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
		{ID: 2, EventType: "amendment", TargetClientEventID: &targetEventID, CorrectedAt: &correctedTime, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
	}
	boundaryStart := parseTime("2026-07-20T04:00:00Z")
	boundaryEnd := boundaryStart.Add(24 * time.Hour)

	// Act
	blocks, err := computeTimeline(events, boundaryStart, boundaryEnd, parseTime("2026-07-20T17:00:00Z"), false)

	// Assert
	if err != nil {
		t.Fatalf("computeTimeline failed: %v", err)
	}
	if blocks[1].StartTime != correctedTime {
		t.Fatalf("expected amended start %v, got %v", correctedTime, blocks[1].StartTime)
	}
}

func TestComputeTimeline_EqualEffectiveTransitionsReturnConflict(t *testing.T) {
	// Arrange
	categoryID := 1
	transitionTime := parseTime("2026-07-20T09:00:00Z")
	events := []DayEvent{
		{ID: 1, EventType: "transition", CategoryID: &categoryID, OccurredAt: transitionTime},
		{ID: 2, EventType: "transition", CategoryID: &categoryID, OccurredAt: transitionTime},
	}
	boundaryStart := parseTime("2026-07-20T04:00:00Z")
	boundaryEnd := boundaryStart.Add(24 * time.Hour)

	// Act
	_, err := computeTimeline(events, boundaryStart, boundaryEnd, parseTime("2026-07-20T17:00:00Z"), false)

	// Assert
	if !errors.Is(err, ErrNonMonotonicTransitions) {
		t.Fatalf("expected non-monotonic transition error, got %v", err)
	}
}
