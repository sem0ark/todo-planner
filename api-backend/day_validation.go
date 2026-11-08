package main

import "sort"

var (
	ErrUnknownCategoryID           = NewBadRequestError("unknown category_id")
	ErrMissingEventCategory        = NewBadRequestError("category_id is required for all events")
	ErrInvalidEventType            = NewBadRequestError("invalid event_type")
	ErrIncompleteAmendment         = NewBadRequestError("amendments require target_client_event_id and corrected_at")
	ErrMissingEventTimestamp       = NewBadRequestError("occurred_at is required")
	ErrMissingClientEventID        = NewBadRequestError("client_event_id is required")
	ErrInvalidActualBlockType      = NewBadRequestError("block_type must be 'actual'")
	ErrActualBlockCategoryRequired = NewBadRequestError("category_id is required for actual blocks")
	ErrInvalidBlockStartTime       = NewBadRequestError("start_at must be a valid UTC timestamp")
	ErrInvalidBlockDuration        = NewBadRequestError("block duration must be non-negative")
	ErrInvalidDayDateRange         = NewBadRequestError("invalid date range")
	ErrDeviceIDRequired            = NewBadRequestError("device_id is required")
	ErrInvalidTimezoneOffset       = NewBadRequestError("timezone_offset_minutes must be between -840 and 840")
	ErrInvalidNoteText             = NewBadRequestError("note text is required and must be at most 512 characters")
)

const minutesPerDay = 24 * 60

type DayRecordTemplateInput struct {
	DayTemplateID *int `json:"day_template_id"`
}

// maxTimezoneOffsetMinutes is the largest real-world UTC offset (UTC+14).
const maxTimezoneOffsetMinutes = 14 * 60

func validateTimezoneOffsetMinutes(offsetMinutes int) error {
	if offsetMinutes < -maxTimezoneOffsetMinutes || offsetMinutes > maxTimezoneOffsetMinutes {
		return ErrInvalidTimezoneOffset
	}
	return nil
}

func validateDateEvents(events []DayEventInput) error {
	for _, event := range events {
		if event.ClientEventID == "" {
			return ErrMissingClientEventID
		}
		if event.EventType != "confirmation" && event.EventType != "transition" && event.EventType != "note" && event.EventType != "amendment" {
			return ErrInvalidEventType
		}
		if (event.EventType == "transition" || event.EventType == "confirmation" || event.EventType == "note") && event.CategoryID == nil {
			return ErrMissingEventCategory
		}
		if event.EventType == "note" && (event.Text == nil || len([]rune(*event.Text)) == 0 || len([]rune(*event.Text)) > 512) {
			return ErrInvalidNoteText
		}
		if event.EventType == "amendment" && (event.TargetClientEventID == "" || event.CorrectedAt == nil) {
			return ErrIncompleteAmendment
		}
		if event.OccurredAt.IsZero() {
			return ErrMissingEventTimestamp
		}
	}
	return nil
}

func validateActualBlocks(blocks []ActualBlockInput) error {
	for _, block := range blocks {
		if block.BlockType != "actual" {
			return ErrInvalidActualBlockType
		}
		if block.CategoryID == nil {
			return ErrActualBlockCategoryRequired
		}
		if block.StartAt.IsZero() {
			return ErrInvalidBlockStartTime
		}
		if block.DurationMinutes < 0 {
			return ErrInvalidBlockDuration
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
		laterStart, laterEnd := laterBlock.StartAt, laterBlock.endAt()
		remainingBlocks := make([]ActualBlockInput, 0, len(resolvedBlocks)+1)

		for _, earlierBlock := range resolvedBlocks {
			earlierStart, earlierEnd := earlierBlock.StartAt, earlierBlock.endAt()
			if !laterEnd.After(earlierStart) || !laterStart.Before(earlierEnd) {
				remainingBlocks = append(remainingBlocks, earlierBlock)
				continue
			}

			if earlierStart.Before(laterStart) {
				leftBlock := earlierBlock
				leftBlock.DurationMinutes = int(laterStart.Sub(earlierStart).Minutes())
				if leftBlock.DurationMinutes > 0 {
					remainingBlocks = append(remainingBlocks, leftBlock)
				}
			}
			if laterEnd.Before(earlierEnd) {
				rightBlock := earlierBlock
				rightBlock.StartAt = laterEnd
				rightBlock.DurationMinutes = int(earlierEnd.Sub(laterEnd).Minutes())
				if rightBlock.DurationMinutes > 0 {
					remainingBlocks = append(remainingBlocks, rightBlock)
				}
			}
		}

		resolvedBlocks = append(remainingBlocks, laterBlock)
	}

	sort.SliceStable(resolvedBlocks, func(leftIndex, rightIndex int) bool {
		return resolvedBlocks[leftIndex].StartAt.Before(resolvedBlocks[rightIndex].StartAt)
	})
	return resolvedBlocks
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
