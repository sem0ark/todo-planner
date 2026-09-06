package main

import "time"

// ComputedBlock is an actual block derived from the event stream.
type ComputedBlock struct {
	CategoryID      *int
	StartTime       time.Time
	DurationMinutes int
}

// computeActualBlocks converts transitions into completed or ongoing blocks.
// Confirmation events do not change the active category and are ignored.
func computeActualBlocks(events []DayEvent, referenceTime time.Time) []ComputedBlock {
	computedBlocks := make([]ComputedBlock, 0)

	for eventIndex, event := range events {
		if event.EventType == "confirmation" {
			continue
		}
		if event.EventType != "transition" {
			continue
		}

		endTime := referenceTime
		for followingIndex := eventIndex + 1; followingIndex < len(events); followingIndex++ {
			if events[followingIndex].EventType == "transition" {
				endTime = events[followingIndex].OccurredAt
				break
			}
		}

		durationMinutes := int(endTime.Sub(event.OccurredAt).Minutes())
		if durationMinutes <= 0 {
			continue
		}

		computedBlocks = append(computedBlocks, ComputedBlock{
			CategoryID:      event.CategoryID,
			StartTime:       event.OccurredAt,
			DurationMinutes: durationMinutes,
		})
	}

	return computedBlocks
}
