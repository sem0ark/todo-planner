package main

import (
	"encoding/json"
	"time"
)

const (
	// DateFormat is the canonical calendar date format used by the API.
	DateFormat = "2006-01-02"
	// ScheduleTimeFormat is the canonical time-of-day format used by the API.
	ScheduleTimeFormat = "15:04:05"
	// TimestampFormat is the canonical UTC timestamp format used by the API.
	TimestampFormat = "2006-01-02T15:04:05Z"
)

// APITimestamp serializes timestamps as UTC values without fractional seconds.
type APITimestamp time.Time

// MarshalJSON implements json.Marshaler for the canonical API timestamp format.
func (timestamp APITimestamp) MarshalJSON() ([]byte, error) {
	formattedTimestamp := time.Time(timestamp).UTC().Format(TimestampFormat)
	return json.Marshal(formattedTimestamp)
}

// UnmarshalJSON implements json.Unmarshaler for the canonical API timestamp format.
func (timestamp *APITimestamp) UnmarshalJSON(data []byte) error {
	var timestampValue string
	if err := json.Unmarshal(data, &timestampValue); err != nil {
		return err
	}

	parsedTimestamp, err := time.Parse(TimestampFormat, timestampValue)
	if err != nil {
		return err
	}
	*timestamp = APITimestamp(parsedTimestamp.UTC())
	return nil
}

// APIScheduleTime serializes a time of day in HH:MM:SS form.
type APIScheduleTime string

// MarshalJSON validates and serializes the canonical API schedule time format.
func (scheduleTime APIScheduleTime) MarshalJSON() ([]byte, error) {
	if _, err := time.Parse(ScheduleTimeFormat, string(scheduleTime)); err != nil {
		return nil, err
	}
	return json.Marshal(string(scheduleTime))
}

// UnmarshalJSON accepts only the canonical HH:MM:SS representation.
func (scheduleTime *APIScheduleTime) UnmarshalJSON(data []byte) error {
	var scheduleTimeValue string
	if err := json.Unmarshal(data, &scheduleTimeValue); err != nil {
		return err
	}

	parsedScheduleTime, err := time.Parse(ScheduleTimeFormat, scheduleTimeValue)
	if err != nil {
		return err
	}
	*scheduleTime = APIScheduleTime(parsedScheduleTime.Format(ScheduleTimeFormat))
	return nil
}
