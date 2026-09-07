package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestAPIScheduleTimeJSON(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{name: "canonical", input: `"08:00:00"`},
		{name: "minutes only", input: `"08:00"`, shouldErr: true},
		{name: "fractional seconds", input: `"08:00:00.000"`, shouldErr: true},
		{name: "timestamp", input: `"2026-09-07T08:00:00Z"`, shouldErr: true},
		{name: "null", input: `null`, shouldErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var value ScheduleTime
			err := json.Unmarshal([]byte(testCase.input), &value)
			if (err != nil) != testCase.shouldErr {
				t.Fatalf("unexpected error state: %v", err)
			}
		})
	}
}

func TestAPIScheduleTimeMarshalJSON(t *testing.T) {
	encoded, err := json.Marshal(mustScheduleTime("08:00:00"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(encoded) != `"08:00:00"` {
		t.Fatalf("unexpected JSON: %s", encoded)
	}
}

func TestScheduleTimeScanAcceptsPostgresTimeValues(t *testing.T) {
	testCases := []struct {
		name           string
		databaseValue  any
		expectedHour   int
		expectedMinute int
		expectedSecond int
	}{
		{
			name:          "time.Time",
			databaseValue: time.Date(0, time.January, 1, 4, 30, 15, 0, time.UTC),
			expectedHour:  4, expectedMinute: 30, expectedSecond: 15,
		},
		{
			name:          "pgtype.Time",
			databaseValue: pgtype.Time{Microseconds: (4*60*60 + 30*60 + 15) * 1_000_000, Valid: true},
			expectedHour:  4, expectedMinute: 30, expectedSecond: 15,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var scheduleTime ScheduleTime
			err := scheduleTime.Scan(testCase.databaseValue)
			if err != nil {
				t.Fatalf("expected database time to scan: %v", err)
			}
			if scheduleTime.Hour() != testCase.expectedHour ||
				scheduleTime.Minute() != testCase.expectedMinute ||
				scheduleTime.Second() != testCase.expectedSecond {
				t.Fatalf("unexpected schedule time: %02d:%02d:%02d", scheduleTime.Hour(), scheduleTime.Minute(), scheduleTime.Second())
			}
		})
	}
}

func TestAPITimestampJSON(t *testing.T) {
	parsedTimestamp := time.Date(2026, time.September, 7, 8, 0, 0, 123000000, time.FixedZone("local", 3600))
	encoded, err := json.Marshal(APITimestamp(parsedTimestamp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(encoded) != `"2026-09-07T07:00:00Z"` {
		t.Fatalf("unexpected JSON: %s", encoded)
	}

	for _, input := range []string{`"2026-09-07T08:00:00.000Z"`, `"2026-09-07T08:00:00+01:00"`, `null`} {
		var value APITimestamp
		if err := json.Unmarshal([]byte(input), &value); err == nil {
			t.Fatalf("expected %s to be rejected", input)
		}
	}
}
