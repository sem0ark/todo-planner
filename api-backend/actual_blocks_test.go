package main

import (
	"errors"
	"testing"
)

func TestComputeTimelineEventConfigurations(t *testing.T) {
	firstCategoryID := 1
	secondCategoryID := 2
	now := parseTime("2026-07-20T17:00:00Z")
	testCases := []struct {
		name              string
		events            []DayEvent
		isPastDay         bool
		expectedTypes     []string
		expectedDurations []int
	}{
		{name: "empty event list", expectedTypes: []string{}, expectedDurations: []int{}},
		{
			name:          "confirmations only",
			events:        []DayEvent{{ID: 1, EventType: "confirmation", OccurredAt: parseTime("2026-07-20T09:00:00Z")}},
			expectedTypes: []string{}, expectedDurations: []int{},
		},
		{
			name: "confirmation before first transition is ignored",
			events: []DayEvent{
				{ID: 1, EventType: "confirmation", OccurredAt: parseTime("2026-07-20T08:00:00Z")},
				{ID: 2, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
			},
			expectedTypes: []string{"actual"}, expectedDurations: []int{30},
		},
		{
			name: "confirmations do not split a block",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "confirmation", OccurredAt: parseTime("2026-07-20T10:00:00Z")},
				{ID: 3, EventType: "transition", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
			},
			expectedTypes: []string{"actual", "actual"}, expectedDurations: []int{180, 30},
		},
		{
			name: "confirmation with different category starts a block",
			events: []DayEvent{
				{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")},
				{ID: 2, EventType: "confirmation", CategoryID: &secondCategoryID, OccurredAt: parseTime("2026-07-20T12:00:00Z")},
			},
			expectedTypes: []string{"actual", "actual"}, expectedDurations: []int{180, 30},
		},
		{
			name:      "past day closes final block",
			events:    []DayEvent{{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-20T09:00:00Z")}},
			isPastDay: true, expectedTypes: []string{"actual"}, expectedDurations: []int{30},
		},
		{
			name:          "events are not filtered by a server day window",
			events:        []DayEvent{{ID: 1, EventType: "transition", CategoryID: &firstCategoryID, OccurredAt: parseTime("2026-07-19T09:00:00Z")}},
			expectedTypes: []string{"actual"}, expectedDurations: []int{30},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			blocks, err := computeTimeline(testCase.events, now, testCase.isPastDay)
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
	}
	blocks, err := computeTimeline(events, parseTime("2026-07-20T17:00:00Z"), false)
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
	_, err := computeTimeline(events, parseTime("2026-07-20T17:00:00Z"), false)
	if !errors.Is(err, ErrNonMonotonicTransitions) {
		t.Fatalf("expected non-monotonic transition error, got %v", err)
	}
}
