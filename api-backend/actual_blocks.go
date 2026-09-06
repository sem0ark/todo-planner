package main

import (
	"errors"
	"sort"
	"time"
)

var ErrNonMonotonicTransitions = errors.New("amendment produces invalid ordering")

type ComputedBlock struct {
	CategoryID      *int
	BlockType       string
	StartTime       time.Time
	DurationMinutes int
	IsOpen          bool
}

type resolvedDayEvent struct {
	event       DayEvent
	effectiveAt time.Time
}

// resolveTimeline applies the latest correction per target, then orders by the
// corrected timestamp and the server insertion id. Events outside the day are
// retained in day_events but do not participate in derived blocks.
func resolveTimeline(events []DayEvent, boundaryStart, boundaryEnd time.Time) []resolvedDayEvent {
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
		if effectiveAt.Before(boundaryStart) || effectiveAt.After(boundaryEnd) {
			continue
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

func computeResolvedBlocks(events []resolvedDayEvent, boundaryStart, boundaryEnd, now time.Time, isPastDay bool) ([]ComputedBlock, error) {
	transitions := make([]resolvedDayEvent, 0)
	for _, event := range events {
		if event.event.EventType == "transition" {
			transitions = append(transitions, event)
		}
	}
	if len(transitions) == 0 {
		return []ComputedBlock{{BlockType: "untracked", StartTime: boundaryStart, DurationMinutes: int(boundaryEnd.Sub(boundaryStart).Minutes())}}, nil
	}

	blocks := make([]ComputedBlock, 0, len(transitions)+1)
	if transitions[0].effectiveAt.After(boundaryStart) {
		blocks = append(blocks, ComputedBlock{BlockType: "untracked", StartTime: boundaryStart, DurationMinutes: int(transitions[0].effectiveAt.Sub(boundaryStart).Minutes())})
	}
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
	end := now
	open := true
	if isPastDay {
		end = boundaryEnd
		open = false
	}
	if !end.Before(last.effectiveAt) {
		blocks = append(blocks, ComputedBlock{
			CategoryID: last.event.CategoryID, BlockType: "actual", StartTime: last.effectiveAt,
			DurationMinutes: int(end.Sub(last.effectiveAt).Minutes()), IsOpen: open,
		})
	}
	return blocks, nil
}

func computeTimeline(events []DayEvent, boundaryStart, boundaryEnd, now time.Time, isPastDay bool) ([]ComputedBlock, error) {
	resolvedEvents := resolveTimeline(events, boundaryStart, boundaryEnd)
	return computeResolvedBlocks(resolvedEvents, boundaryStart, boundaryEnd, now, isPastDay)
}
