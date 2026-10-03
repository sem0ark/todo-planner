package main

import (
	"sort"
	"time"
)

var ErrNonMonotonicTransitions = NewBadRequestError("amendment produces invalid ordering")

type ComputedBlock struct {
	CategoryID      *int
	BlockType       string
	StartTime       time.Time
	DurationMinutes int
}

type resolvedDayEvent struct {
	event       DayEvent
	effectiveAt time.Time
}

// resolveTimeline applies the latest correction per target, then orders by the
// corrected timestamp and the server insertion id.
func resolveTimeline(events []DayEvent) []resolvedDayEvent {
	amendmentsByTarget := make(map[string]DayEvent)
	for _, event := range events {
		if event.EventType != "amendment" || event.TargetClientEventID == nil {
			continue
		}
		target := *event.TargetClientEventID
		current, exists := amendmentsByTarget[target]
		if !exists || event.OccurredAt.After(current.OccurredAt) ||
			(event.OccurredAt.Equal(current.OccurredAt) && event.ID > current.ID) {
			amendmentsByTarget[target] = event
		}
	}

	resolvedEvents := make([]resolvedDayEvent, 0, len(events))
	for _, event := range events {
		if event.EventType == "amendment" {
			continue
		}
		effectiveAt := event.OccurredAt
		if event.ClientEventID != nil {
			if amendment, exists := amendmentsByTarget[*event.ClientEventID]; exists && amendment.CorrectedAt != nil {
				effectiveAt = *amendment.CorrectedAt
			}
		}
		resolvedEvents = append(resolvedEvents, resolvedDayEvent{event: event, effectiveAt: effectiveAt})
	}

	sort.SliceStable(resolvedEvents, func(leftIndex, rightIndex int) bool {
		left := resolvedEvents[leftIndex]
		right := resolvedEvents[rightIndex]
		if left.effectiveAt.Equal(right.effectiveAt) {
			return left.event.ID < right.event.ID
		}
		return left.effectiveAt.Before(right.effectiveAt)
	})
	return resolvedEvents
}

func computeResolvedBlocks(events []resolvedDayEvent) ([]ComputedBlock, error) {
	transitions := make([]resolvedDayEvent, 0)
	var currentCategoryID *int
	for _, event := range events {
		isBoundary := event.event.EventType == "transition"
		if event.event.EventType == "confirmation" && event.event.CategoryID != nil {
			isBoundary = currentCategoryID == nil || *currentCategoryID != *event.event.CategoryID
		}
		if isBoundary {
			transitions = append(transitions, event)
			currentCategoryID = event.event.CategoryID
		}
	}
	if len(transitions) == 0 {
		return []ComputedBlock{}, nil
	}

	blocks := make([]ComputedBlock, 0, len(transitions)+1)
	for index := 0; index < len(transitions)-1; index++ {
		start := transitions[index].effectiveAt
		end := transitions[index+1].effectiveAt
		if !end.After(start) {
			return nil, ErrNonMonotonicTransitions
		}
		blocks = append(blocks, ComputedBlock{
			CategoryID: transitions[index].event.CategoryID, BlockType: "actual", StartTime: start,
			DurationMinutes: int(end.Sub(start).Minutes()),
		})
	}

	last := transitions[len(transitions)-1]
	latestConfirmationAt := latestConfirmationAfter(events, last.effectiveAt)
	if latestConfirmationAt != nil {
		blocks = append(blocks, ComputedBlock{
			CategoryID: last.event.CategoryID, BlockType: "actual", StartTime: last.effectiveAt,
			DurationMinutes: int(latestConfirmationAt.Sub(last.effectiveAt).Minutes()),
		})
	}
	return blocks, nil
}

func latestConfirmationAfter(events []resolvedDayEvent, start time.Time) *time.Time {
	for eventIndex := len(events) - 1; eventIndex >= 0; eventIndex-- {
		event := events[eventIndex]
		if !event.effectiveAt.After(start) {
			return nil
		}
		if event.event.EventType == "confirmation" {
			return &event.effectiveAt
		}
	}
	return nil
}

func computeTimeline(events []DayEvent) ([]ComputedBlock, error) {
	resolvedEvents := resolveTimeline(events)
	return computeResolvedBlocks(resolvedEvents)
}
