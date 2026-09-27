package main

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
	ErrActualBlocksOverlap         = NewBadRequestError("actual blocks must not overlap")
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
		if event.EventType == "transition" && event.CategoryID == nil {
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
	previousEndSecond := 0
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
		if block.StartTime.IsZero() {
			return ErrInvalidBlockStartTime
		}
		if block.DurationMinutes < 0 {
			return ErrInvalidBlockDuration
		}
		if blockExceedsDay(block.StartTime, block.DurationMinutes) {
			return ErrBlockExceedsDay
		}
		startSecond := scheduleTimeSecondOfDay(block.StartTime)
		if index > 0 && startSecond < previousEndSecond {
			return ErrActualBlocksOverlap
		}
		previousEndSecond = startSecond + block.DurationMinutes*60
	}
	return nil
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
