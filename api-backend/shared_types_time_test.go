package main

import (
	"encoding/json"
	"testing"
	"time"
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
			var value APIScheduleTime
			err := json.Unmarshal([]byte(testCase.input), &value)
			if (err != nil) != testCase.shouldErr {
				t.Fatalf("unexpected error state: %v", err)
			}
		})
	}
}

func TestAPIScheduleTimeMarshalJSON(t *testing.T) {
	encoded, err := json.Marshal(APIScheduleTime("08:00:00"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(encoded) != `"08:00:00"` {
		t.Fatalf("unexpected JSON: %s", encoded)
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
