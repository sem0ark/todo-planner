package main

import (
	"sort"
	"time"
)

var (
	ErrUnknownCategoryID           = NewBadRequestError("unknown category_id")
	ErrMissingEventCategory        = NewBadRequestError("category_id is required for all events")
	ErrInvalidEventType            = NewBadRequestError("invalid event_type")
	ErrIncompleteAmendment         = NewBadRequestError("amendments require target_client_event_id and corrected_at")
	ErrMissingEventTimestamp       = NewBadRequestError("occurred_at is required")
	ErrUnsortedEvents              = NewBadRequestError("events must be in chronological order")
	ErrMissingClientEventID        = NewBadRequestError("client_event_id is required")
	ErrInvalidActualBlockType      = NewBadRequestError("block_type must be 'actual' or 'blank'")
	ErrActualBlockCategoryRequired = NewBadRequestError("category_id is required for actual blocks")
	ErrBlankBlockCategoryForbidden = NewBadRequestError("category_id must be null for blank blocks")
	ErrInvalidBlockStartTime       = NewBadRequestError("start_time must be a valid time")
	ErrInvalidBlockDuration        = NewBadRequestError("block duration must be non-negative")
	ErrBlockExceedsDay             = NewBadRequestError("block must end by 24:00")
	ErrInvalidDayDateRange         = NewBadRequestError("invalid date range")
	ErrDeviceIDRequired            = NewBadRequestError("device_id is required")
	ErrMissingLocalTimestamp       = NewBadRequestError("occurred_at_local is required")
	ErrInvalidLocalTimestamp       = NewBadRequestError("occurred_at_local must be RFC3339 with a UTC offset")
	ErrEventLocalDateMismatch      = NewBadRequestError("occurred_at_local date does not match the requested day")
	ErrAmendmentOffsetMismatch     = NewBadRequestError("amendment offset must match the target event's recorded offset")
)

const minutesPerDay = 24 * 60

type DayRecordBlocksInput struct {
	ActualBlocks        []ActualBlockInput `json:"actual_blocks"`
	ClientOffsetMinutes *int               `json:"client_offset_minutes"`
}

type DayRecordTemplateInput struct {
	DayTemplateID *int `json:"day_template_id"`
}

type ActualBlockInput struct {
	CategoryID      *int         `json:"category_id"`
	BlockType       string       `json:"block_type"`
	StartTime       ScheduleTime `json:"-"`
	DurationMinutes int          `json:"duration_minutes"`
}

func validateDayEvents(events []DayEventInput) error {
	for index, event := range events {
		if event.EventType != "confirmation" && event.EventType != "transition" && event.EventType != "amendment" {
			return ErrInvalidEventType
		}
		if (event.EventType == "transition" || event.EventType == "confirmation") && event.CategoryID == nil {
			return ErrMissingEventCategory
		}
		if event.EventType == "amendment" && (event.TargetClientEventID == "" || event.CorrectedAt == nil) {
			return ErrIncompleteAmendment
		}
		if event.OccurredAt.IsZero() {
			return ErrMissingEventTimestamp
		}
		if event.OccurredAtLocal == "" {
			return ErrMissingLocalTimestamp
		}
		if _, err := extractOffsetMinutes(event.OccurredAtLocal); err != nil {
			return ErrInvalidLocalTimestamp
		}
		if index > 0 && event.OccurredAt.Before(events[index-1].OccurredAt) {
			return ErrUnsortedEvents
		}
		if event.EventType == "amendment" {
			if event.CorrectedAtLocal == nil {
				return ErrIncompleteAmendment
			}
			if _, err := extractOffsetMinutes(*event.CorrectedAtLocal); err != nil {
				return ErrInvalidLocalTimestamp
			}
		}
	}
	return nil
}

func validateDateEvents(events []DayEventInput) error {
	if err := validateDayEvents(events); err != nil {
		return err
	}
	for _, event := range events {
		if event.ClientEventID == "" {
			return ErrMissingClientEventID
		}
	}
	return nil
}

func validateActualBlocks(blocks []ActualBlockInput) error {
	for _, block := range blocks {
		if block.BlockType != "actual" && block.BlockType != "blank" {
			return ErrInvalidActualBlockType
		}
		if block.BlockType == "actual" && block.CategoryID == nil {
			return ErrActualBlockCategoryRequired
		}
		if block.BlockType == "blank" && block.CategoryID != nil {
			return ErrBlankBlockCategoryForbidden
		}
		if block.StartTime.IsZero() {
			return ErrInvalidBlockStartTime
		}
		if block.DurationMinutes < 0 {
			return ErrInvalidBlockDuration
		}
		if blockExceedsDay(block.StartTime, block.DurationMinutes) {
			return ErrBlockExceedsDay
		}
	}
	return nil
}

// resolveOverlappingActualBlocks applies posted blocks in order. Each later
// block replaces the covered portions of earlier blocks, splitting an earlier
// block when the later block is in its middle.
func resolveOverlappingActualBlocks(blocks []ActualBlockInput) []ActualBlockInput {
	resolvedBlocks := make([]ActualBlockInput, 0, len(blocks))
	for _, laterBlock := range blocks {
		laterStartSecond := scheduleTimeSecondOfDay(laterBlock.StartTime)
		laterEndSecond := laterStartSecond + laterBlock.DurationMinutes*60
		remainingBlocks := make([]ActualBlockInput, 0, len(resolvedBlocks)+1)

		for _, earlierBlock := range resolvedBlocks {
			earlierStartSecond := scheduleTimeSecondOfDay(earlierBlock.StartTime)
			earlierEndSecond := earlierStartSecond + earlierBlock.DurationMinutes*60
			if laterEndSecond <= earlierStartSecond || laterStartSecond >= earlierEndSecond {
				remainingBlocks = append(remainingBlocks, earlierBlock)
				continue
			}

			if earlierStartSecond < laterStartSecond {
				leftBlock := earlierBlock
				leftBlock.DurationMinutes = (laterStartSecond - earlierStartSecond) / 60
				if leftBlock.DurationMinutes > 0 {
					remainingBlocks = append(remainingBlocks, leftBlock)
				}
			}
			if laterEndSecond < earlierEndSecond {
				rightBlock := earlierBlock
				rightBlock.StartTime = scheduleTimeFromSeconds(laterEndSecond)
				rightBlock.DurationMinutes = (earlierEndSecond - laterEndSecond) / 60
				if rightBlock.DurationMinutes > 0 {
					remainingBlocks = append(remainingBlocks, rightBlock)
				}
			}
		}

		resolvedBlocks = append(remainingBlocks, laterBlock)
	}

	sort.SliceStable(resolvedBlocks, func(leftIndex, rightIndex int) bool {
		return scheduleTimeSecondOfDay(resolvedBlocks[leftIndex].StartTime) <
			scheduleTimeSecondOfDay(resolvedBlocks[rightIndex].StartTime)
	})
	return resolvedBlocks
}

func scheduleTimeFromSeconds(totalSeconds int) ScheduleTime {
	return ScheduleTime(time.Date(0, time.January, 1, 0, 0, totalSeconds, 0, time.UTC))
}

func blockExceedsDay(startTime ScheduleTime, durationMinutes int) bool {
	return scheduleTimeSecondOfDay(startTime)+durationMinutes*60 > minutesPerDay*60
}

func scheduleTimeSecondOfDay(scheduleTime ScheduleTime) int {
	return scheduleTime.Hour()*60*60 + scheduleTime.Minute()*60 + scheduleTime.Second()
}

func isValidCalendarDate(date string) bool {
	_, err := parseCalendarDate(date)
	return err == nil
}
