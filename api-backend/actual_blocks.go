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

type timelineObservation struct {
	occurredAt time.Time
	categoryID *int
}

type timelineBlock struct {
	categoryID *int
	start      time.Time
	end        time.Time
}

type resolvedDayEvent struct {
	event       DayEvent
	effectiveAt time.Time
}

// resolveTimeline applies the latest correction per target, then orders by the
// corrected timestamp and the server insertion id.
func resolveTimeline(events []DayEvent) []resolvedDayEvent {
	amendmentsByTarget := make(map[string]DayEvent)
	eventsByClientID := make(map[string]struct{})
	for _, event := range events {
		if event.ClientEventID != nil {
			eventsByClientID[*event.ClientEventID] = struct{}{}
		}
	}
	for _, event := range events {
		if event.EventType != "amendment" || event.TargetClientEventID == nil {
			continue
		}
		if _, exists := eventsByClientID[*event.TargetClientEventID]; !exists {
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

func observationsFromEvents(events []resolvedDayEvent) []timelineObservation {
	observations := make([]timelineObservation, 0, len(events))
	for _, event := range events {
		if event.event.EventType != "transition" && event.event.EventType != "confirmation" {
			continue
		}
		if event.event.CategoryID == nil {
			continue
		}
		observations = append(observations, timelineObservation{
			occurredAt: event.effectiveAt,
			categoryID: event.event.CategoryID,
		})
	}
	return observations
}

func calculateTimelineBlocks(observations []timelineObservation) ([]timelineBlock, error) {
	observations = collapseEqualTimestampObservations(observations)
	blocks := make([]timelineBlock, 0, len(observations))
	for index := 0; index < len(observations)-1; index++ {
		start := observations[index]
		end := observations[index+1]
		if !end.occurredAt.After(start.occurredAt) {
			return nil, ErrNonMonotonicTransitions
		}
		blocks = append(blocks, timelineBlock{
			categoryID: start.categoryID,
			start:      start.occurredAt,
			end:        end.occurredAt,
		})
	}
	return blocks, nil
}

// collapseEqualTimestampObservations keeps the last observation at a timestamp.
// Resolved events are sorted by timestamp and server ID, so the last observation
// is the deterministic final state for events recorded at the same instant.
func collapseEqualTimestampObservations(observations []timelineObservation) []timelineObservation {
	if len(observations) < 2 {
		return observations
	}

	collapsedObservations := make([]timelineObservation, 0, len(observations))
	for _, observation := range observations {
		lastIndex := len(collapsedObservations) - 1
		if lastIndex >= 0 && observation.occurredAt.Equal(collapsedObservations[lastIndex].occurredAt) {
			collapsedObservations[lastIndex] = observation
			continue
		}
		collapsedObservations = append(collapsedObservations, observation)
	}
	return collapsedObservations
}

func normalizeTimelineBlocks(blocks []timelineBlock) []timelineBlock {
	longBlocks := make([]timelineBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.end.Sub(block.start) < minimumActualBlockDuration {
			continue
		}
		longBlocks = append(longBlocks, block)
	}

	mergedBlocks := make([]timelineBlock, 0, len(longBlocks))
	for _, block := range longBlocks {
		if len(mergedBlocks) == 0 {
			mergedBlocks = append(mergedBlocks, block)
			continue
		}
		previousIndex := len(mergedBlocks) - 1
		previous := &mergedBlocks[previousIndex]
		if sameCategory(previous.categoryID, block.categoryID) &&
			block.start.Sub(previous.end) <= minimumActualBlockDuration {
			if block.end.After(previous.end) {
				previous.end = block.end
			}
			continue
		}
		mergedBlocks = append(mergedBlocks, block)
	}
	return mergedBlocks
}

func trimTimelineBlocks(blocks []timelineBlock, checkpoint time.Time) []timelineBlock {
	trimmedBlocks := make([]timelineBlock, 0, len(blocks))
	for _, block := range blocks {
		if !block.end.After(checkpoint) {
			continue
		}
		if block.start.Before(checkpoint) {
			block.start = checkpoint
		}
		if block.end.After(block.start) {
			trimmedBlocks = append(trimmedBlocks, block)
		}
	}
	return trimmedBlocks
}

func sameCategory(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func computedBlocksFromTimeline(blocks []timelineBlock) []ComputedBlock {
	computedBlocks := make([]ComputedBlock, 0, len(blocks))
	for _, block := range blocks {
		computedBlocks = append(computedBlocks, ComputedBlock{
			CategoryID:      block.categoryID,
			BlockType:       "actual",
			StartTime:       block.start,
			DurationMinutes: int(block.end.Sub(block.start).Minutes()),
		})
	}
	return computedBlocks
}

func computeResolvedBlocks(events []resolvedDayEvent) ([]ComputedBlock, error) {
	observations := observationsFromEvents(events)
	rawBlocks, err := calculateTimelineBlocks(observations)
	if err != nil {
		return nil, err
	}
	return computedBlocksFromTimeline(normalizeTimelineBlocks(rawBlocks)), nil
}

func computeTimeline(events []DayEvent) ([]ComputedBlock, error) {
	resolvedEvents := resolveTimeline(events)
	return computeResolvedBlocks(resolvedEvents)
}

// computeIncrementalTimeline calculates the complete event timeline, then
// returns only the portion after the latest saved actual block.
func computeIncrementalTimeline(events []DayEvent, existingBlocks []ActualBlock, calendarDate CalendarDate) ([]ComputedBlock, error) {
	checkpoint, hasCheckpoint := actualBlockCheckpoint(existingBlocks, calendarDate)
	if !hasCheckpoint {
		checkpoint = time.Date(calendarDate.Year(), calendarDate.Month(), calendarDate.Day(), 0, 0, 0, 0, time.UTC)
	}

	resolvedEvents := resolveTimeline(events)
	observations := observationsFromEvents(resolvedEvents)

	rawBlocks, err := calculateTimelineBlocks(observations)
	if err != nil {
		return nil, err
	}
	trimmedBlocks := trimTimelineBlocks(rawBlocks, checkpoint)
	return computedBlocksFromTimeline(normalizeTimelineBlocks(trimmedBlocks)), nil
}

func actualBlockCheckpoint(blocks []ActualBlock, calendarDate CalendarDate) (time.Time, bool) {
	var latestBlock *ActualBlock
	for blockIndex := range blocks {
		block := &blocks[blockIndex]
		if block.BlockType != "actual" {
			continue
		}
		if latestBlock == nil || time.Time(block.StartTime).After(time.Time(latestBlock.StartTime)) {
			latestBlock = block
		}
	}
	if latestBlock == nil {
		return time.Time{}, false
	}
	start := time.Date(
		calendarDate.Year(), calendarDate.Month(), calendarDate.Day(),
		latestBlock.StartTime.Hour(), latestBlock.StartTime.Minute(), latestBlock.StartTime.Second(),
		0, time.UTC,
	)
	return start.Add(time.Duration(latestBlock.DurationMinutes) * time.Minute), true
}
