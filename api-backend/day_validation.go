package main

import (
	"errors"
	"time"
)

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
)

type DayRecordBlocksInput struct {
	ActualBlocks []ActualBlockInput `json:"actual_blocks"`
}

type DayRecordTemplateInput struct {
	DayTemplateID *int `json:"day_template_id"`
}

type ActualBlockInput struct {
	CategoryID      *int   `json:"category_id"`
	BlockType       string `json:"block_type"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
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
		parsedTime, err := time.Parse("15:04:05", block.StartTime)
		if err != nil {
			parsedTime, err = time.Parse("15:04", block.StartTime)
		}
		if err != nil || parsedTime.Second() != 0 {
			return ErrInvalidBlockStartTime
		}
		minuteOfDay := parsedTime.Hour()*60 + parsedTime.Minute()
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

func isValidCalendarDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}
