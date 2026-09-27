package main

import "testing"

func TestExtractOffsetMinutes(t *testing.T) {
	testCases := []struct {
		name      string
		timestamp string
		expected  int
	}{
		{name: "positive offset", timestamp: "2026-09-26T14:30:00+02:00", expected: 120},
		{name: "negative offset", timestamp: "2026-09-26T08:30:00-05:00", expected: -300},
		{name: "quarter hour offset", timestamp: "2026-09-26T20:15:00+05:45", expected: 345},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := extractOffsetMinutes(testCase.timestamp)
			if err != nil {
				t.Fatalf("extractOffsetMinutes failed: %v", err)
			}
			if actual != testCase.expected {
				t.Fatalf("expected %d minutes, got %d", testCase.expected, actual)
			}
		})
	}
}

func TestExtractOffsetMinutesRejectsUTCDesignator(t *testing.T) {
	if _, err := extractOffsetMinutes("2026-09-26T12:30:00Z"); err == nil {
		t.Fatal("expected UTC designator to be rejected")
	}
}

func TestValidateDayEventsRequiresOffsetPreservingTimestamp(t *testing.T) {
	categoryID := 1
	event := DayEventInput{
		ClientEventID: "event-1",
		EventType:     "transition",
		CategoryID:    &categoryID,
		OccurredAt:    parseTime("2026-09-26T12:30:00Z"),
	}
	if err := validateDateEvents([]DayEventInput{event}); err != ErrMissingLocalTimestamp {
		t.Fatalf("expected missing local timestamp, got %v", err)
	}

	event.OccurredAtLocal = "2026-09-26T14:30:00Z"
	if err := validateDateEvents([]DayEventInput{event}); err != ErrInvalidLocalTimestamp {
		t.Fatalf("expected invalid local timestamp, got %v", err)
	}
}

func TestLocalCalendarDateUsesEmbeddedOffset(t *testing.T) {
	calendarDate, err := localCalendarDate("2026-09-26T00:15:00-05:00")
	if err != nil {
		t.Fatalf("localCalendarDate failed: %v", err)
	}
	if calendarDate != mustCalendarDate("2026-09-26") {
		t.Fatalf("expected local date 2026-09-26, got %s", calendarDate)
	}
}

func TestValidateAmendmentOffset(t *testing.T) {
	matchingError := validateAmendmentOffset(
		"2026-09-26T14:30:00+02:00",
		"2026-09-26T15:00:00+02:00",
	)
	if matchingError != nil {
		t.Fatalf("matching offsets rejected: %v", matchingError)
	}

	mismatchError := validateAmendmentOffset(
		"2026-09-26T14:30:00+02:00",
		"2026-09-26T07:00:00-05:00",
	)
	if mismatchError != ErrAmendmentOffsetMismatch {
		t.Fatalf("expected amendment offset mismatch, got %v", mismatchError)
	}
}

func TestToPublicDayRecordShiftsActualStartsOnly(t *testing.T) {
	offsetMinutes := 120
	categoryID := 4
	record := DayRecord{
		CalendarDate:          mustCalendarDate("2026-09-26"),
		TimezoneOffsetMinutes: &offsetMinutes,
		TimezoneOffsetLocked:  true,
		ActualBlocks: []ActualBlock{{
			CategoryID:      &categoryID,
			BlockType:       "actual",
			StartTime:       mustScheduleTime("12:30:00"),
			DurationMinutes: 30,
		}},
	}

	publicRecord := toPublicDayRecord(&record)
	if publicRecord.Actual[0].StartTime != mustScheduleTime("14:30:00") {
		t.Fatalf("expected shifted actual start, got %v", publicRecord.Actual[0].StartTime)
	}
	if publicRecord.TimezoneOffsetMinutes == nil || *publicRecord.TimezoneOffsetMinutes != 120 {
		t.Fatalf("expected offset metadata to be preserved: %+v", publicRecord)
	}
}
