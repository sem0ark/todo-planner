package main

import (
	"fmt"
	"strings"
	"time"
)

func extractOffsetMinutes(localTimestamp string) (int, error) {
	if len(localTimestamp) < len("2006-01-02T15:04:05+00:00") {
		return 0, fmt.Errorf("timestamp has no numeric UTC offset")
	}
	offset := localTimestamp[len(localTimestamp)-6:]
	if (offset[0] != '+' && offset[0] != '-') || offset[3] != ':' {
		return 0, fmt.Errorf("timestamp has no numeric UTC offset")
	}
	parsed, err := time.Parse(time.RFC3339, localTimestamp)
	if err != nil || strings.HasSuffix(localTimestamp, "Z") {
		return 0, fmt.Errorf("timestamp must contain a numeric UTC offset")
	}
	_, seconds := parsed.Zone()
	return seconds / 60, nil
}

func localCalendarDate(localTimestamp string) (CalendarDate, error) {
	parsed, err := time.Parse(time.RFC3339, localTimestamp)
	if err != nil {
		return CalendarDate{}, err
	}
	return CalendarDate(time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)), nil
}

func validateAmendmentOffset(targetLocal, correctedLocal string) error {
	targetOffset, err := extractOffsetMinutes(targetLocal)
	if err != nil {
		return ErrInvalidLocalTimestamp
	}
	correctedOffset, err := extractOffsetMinutes(correctedLocal)
	if err != nil {
		return ErrInvalidLocalTimestamp
	}
	if targetOffset != correctedOffset {
		return ErrAmendmentOffsetMismatch
	}
	return nil
}
