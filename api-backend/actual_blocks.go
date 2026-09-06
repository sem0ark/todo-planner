package main

import (
	"sort"
	"time"
)

// ComputedBlock is an actual block derived from the event stream.
type ComputedBlock struct {
	CategoryID      *int
	StartTime       time.Time
	DurationMinutes int
	IsOpen          bool
}

// computeActualBlocks converts transitions into completed or ongoing blocks.
// Confirmation events do not change the active category and are ignored.
func computeActualBlocks(events []DayEvent, referenceTime time.Time) []ComputedBlock {
	events = applyEventAmendments(events)
	computedBlocks := make([]ComputedBlock, 0)

	for eventIndex, event := range events {
		if event.EventType == "confirmation" {
			continue
		}
		if event.EventType != "transition" {
			continue
		}

		endTime := referenceTime
		isOpen := true
		for followingIndex := eventIndex + 1; followingIndex < len(events); followingIndex++ {
			if events[followingIndex].EventType == "transition" {
				endTime = events[followingIndex].OccurredAt
				isOpen = false
				break
			}
		}

		durationMinutes := int(endTime.Sub(event.OccurredAt).Minutes())
		if durationMinutes < 0 || (durationMinutes == 0 && !isOpen) {
			continue
		}

		computedBlocks = append(computedBlocks, ComputedBlock{
			CategoryID:      event.CategoryID,
			StartTime:       event.OccurredAt,
			DurationMinutes: durationMinutes,
			IsOpen:          isOpen,
		})
	}

	return computedBlocks
}

func applyEventAmendments(events []DayEvent) []DayEvent {
	correctedEvents := make([]DayEvent, 0, len(events))
	for _, event := range events {
		if event.EventType == "amendment" {
			continue
		}
		correctedEvents = append(correctedEvents, event)
	}
	for _, amendment := range events {
		if amendment.EventType != "amendment" {
			continue
		}
		for eventIndex := range correctedEvents {
			if correctedEvents[eventIndex].ClientEventID == nil || amendment.TargetClientEventID == nil || *correctedEvents[eventIndex].ClientEventID != *amendment.TargetClientEventID {
				continue
			}
			if amendment.CorrectedAt != nil {
				correctedEvents[eventIndex].OccurredAt = *amendment.CorrectedAt
			}
			if amendment.CategoryID != nil {
				correctedEvents[eventIndex].CategoryID = amendment.CategoryID
			}
		}
	}
	sort.SliceStable(correctedEvents, func(leftIndex, rightIndex int) bool {
		return correctedEvents[leftIndex].OccurredAt.Before(correctedEvents[rightIndex].OccurredAt)
	})
	return correctedEvents
}
