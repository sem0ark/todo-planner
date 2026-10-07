package main

import (
	"strings"
	"testing"
	"time"
)

func TestValidateDayEventsNoteText(t *testing.T) {
	categoryID := 1
	baseEvent := DayEventInput{
		ClientEventID:   "note-1",
		EventType:       "note",
		CategoryID:      &categoryID,
		OccurredAt:      time.Date(2026, time.July, 20, 9, 0, 0, 0, time.UTC),
		OccurredAtLocal: "2026-07-20T09:00:00+00:00",
	}

	validText := strings.Repeat("a", 512)
	validEvent := baseEvent
	validEvent.Text = &validText
	if err := validateDateEvents([]DayEventInput{validEvent}); err != nil {
		t.Fatalf("expected 512-character note to be valid, got %v", err)
	}

	tooLongText := strings.Repeat("a", 513)
	tooLongEvent := baseEvent
	tooLongEvent.Text = &tooLongText
	if err := validateDateEvents([]DayEventInput{tooLongEvent}); err != ErrInvalidNoteText {
		t.Fatalf("expected oversized note error, got %v", err)
	}

	missingTextEvent := baseEvent
	if err := validateDateEvents([]DayEventInput{missingTextEvent}); err != ErrInvalidNoteText {
		t.Fatalf("expected missing note text error, got %v", err)
	}
}
