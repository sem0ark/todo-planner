package main

import "errors"

var (
	ErrUnknownCategoryID           = errors.New("unknown category_id")
	ErrMissingEventCategory        = errors.New("category_id is required for all events")
	ErrInvalidEventType            = errors.New("invalid event_type")
	ErrIncompleteAmendment         = errors.New("amendments require target_client_event_id and corrected_at")
	ErrMissingEventTimestamp       = errors.New("occurred_at is required")
	ErrUnsortedEvents              = errors.New("events must be in chronological order")
	ErrMissingClientEventID        = errors.New("client_event_id is required")
	ErrInvalidActualBlockType      = errors.New("block_type must be 'actual' or 'blank'")
	ErrActualBlockCategoryRequired = errors.New("category_id is required for actual blocks")
	ErrBlankBlockCategoryForbidden = errors.New("category_id must be null for blank blocks")
	ErrInvalidBlockStartTime       = errors.New("start_time must be a valid time")
	ErrInvalidBlockGranularity     = errors.New("blocks must use 15-minute increments and last at least 30 minutes")
	ErrBlockExceedsDay             = errors.New("block must end by 24:00")
	ErrActualBlocksOverlap         = errors.New("actual blocks must not overlap")
	ErrInvalidDayDateRange         = errors.New("invalid date range")
	ErrDeviceIDRequired            = errors.New("device_id is required")
)

const minutesPerDay = 24 * 60

type DayRecordBlocksInput struct {
	ActualBlocks []ActualBlockInput `json:"actual_blocks"`
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
		if event.EventType == "transition" && event.CategoryID == nil {
			return ErrMissingEventCategory
		}
		if event.EventType == "amendment" && (event.TargetClientEventID == "" || event.CorrectedAt == nil) {
			return ErrIncompleteAmendment
		}
		if event.OccurredAt.IsZero() {
			return ErrMissingEventTimestamp
		}
		if index > 0 && event.OccurredAt.Before(events[index-1].OccurredAt) {
			return ErrUnsortedEvents
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
	previousEndMinute := 0
	for index, block := range blocks {
		if block.BlockType != "actual" && block.BlockType != "blank" {
			return ErrInvalidActualBlockType
		}
		if block.BlockType == "actual" && block.CategoryID == nil {
			return ErrActualBlockCategoryRequired
		}
		if block.BlockType == "blank" && block.CategoryID != nil {
			return ErrBlankBlockCategoryForbidden
		}
		if block.StartTime.IsZero() || block.StartTime.Second() != 0 {
			return ErrInvalidBlockStartTime
		}
		minuteOfDay := block.StartTime.Hour()*60 + block.StartTime.Minute()
		if minuteOfDay%15 != 0 || block.DurationMinutes < 30 || block.DurationMinutes%15 != 0 {
			return ErrInvalidBlockGranularity
		}
		if blockExceedsDay(block.StartTime, block.DurationMinutes) {
			return ErrBlockExceedsDay
		}
		if index > 0 && minuteOfDay < previousEndMinute {
			return ErrActualBlocksOverlap
		}
		previousEndMinute = minuteOfDay + block.DurationMinutes
	}
	return nil
}

func blockExceedsDay(startTime ScheduleTime, durationMinutes int) bool {
	startMinute := startTime.Hour()*60 + startTime.Minute()
	return startMinute+durationMinutes > minutesPerDay
}

func isValidCalendarDate(date string) bool {
	_, err := parseCalendarDate(date)
	return err == nil
}
