package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	DateFormat         = "2006-01-02"
	ScheduleTimeFormat = "15:04:05"
	TimestampFormat    = "2006-01-02T15:04:05Z"
)

// CalendarDate is a calendar date with controlled API serialization.
type CalendarDate time.Time

var _ sql.Scanner = (*CalendarDate)(nil)
var _ driver.Valuer = CalendarDate{}

func (date *CalendarDate) Scan(source any) error {
	switch value := source.(type) {
	case time.Time:
		*date = CalendarDate(value)
		return nil
	case pgtype.Date:
		if !value.Valid {
			return errors.New("invalid null calendar date")
		}
		*date = CalendarDate(value.Time)
		return nil
	case string:
		parsedDate, err := parseCalendarDate(value)
		if err != nil {
			return err
		}
		*date = parsedDate
		return nil
	case []byte:
		parsedDate, err := parseCalendarDate(string(value))
		if err != nil {
			return err
		}
		*date = parsedDate
		return nil
	default:
		return fmt.Errorf("cannot scan %T into CalendarDate", source)
	}
}

func (date CalendarDate) Value() (driver.Value, error) {
	return pgtype.Date{Time: time.Time(date), Valid: true}, nil
}

func (date CalendarDate) Before(other CalendarDate) bool {
	return time.Time(date).Before(time.Time(other))
}
func (date CalendarDate) After(other CalendarDate) bool {
	return time.Time(date).After(time.Time(other))
}
func (date CalendarDate) AddDate(years, months, days int) CalendarDate {
	return CalendarDate(time.Time(date).AddDate(years, months, days))
}
func (date CalendarDate) Sub(other CalendarDate) time.Duration {
	return time.Time(date).Sub(time.Time(other))
}
func (date CalendarDate) Format(layout string) string { return time.Time(date).Format(layout) }
func (date CalendarDate) String() string              { return date.Format(DateFormat) }
func (date CalendarDate) Weekday() time.Weekday       { return time.Time(date).Weekday() }
func (date CalendarDate) Year() int                   { return time.Time(date).Year() }
func (date CalendarDate) Month() time.Month           { return time.Time(date).Month() }
func (date CalendarDate) Day() int                    { return time.Time(date).Day() }

// ScheduleTime is a time of day with controlled API serialization.
type ScheduleTime time.Time

var _ sql.Scanner = (*ScheduleTime)(nil)
var _ driver.Valuer = ScheduleTime{}

func (scheduleTime *ScheduleTime) Scan(source any) error {
	switch value := source.(type) {
	case nil:
		return errors.New("invalid null schedule time")
	case time.Time:
		*scheduleTime = ScheduleTime(value)
		return nil
	case pgtype.Time:
		if !value.Valid {
			return errors.New("invalid null schedule time")
		}
		*scheduleTime = ScheduleTime(time.Unix(0, value.Microseconds*1000).UTC())
		return nil
	case string:
		parsedTime, err := parseDatabaseScheduleTime(value)
		if err != nil {
			return err
		}
		*scheduleTime = parsedTime
		return nil
	case []byte:
		parsedTime, err := parseDatabaseScheduleTime(string(value))
		if err != nil {
			return err
		}
		*scheduleTime = parsedTime
		return nil
	default:
		return fmt.Errorf("cannot scan %T into ScheduleTime", source)
	}
}

func parseDatabaseScheduleTime(value string) (ScheduleTime, error) {
	parsedTime, err := parseScheduleTime(value)
	if err == nil {
		return parsedTime, nil
	}
	for _, layout := range []string{
		"15:04:05.999999999",
		"15:04:05Z07:00",
		"15:04:05.999999999Z07:00",
		time.RFC3339Nano,
	} {
		parsedTimestamp, parseError := time.Parse(layout, value)
		if parseError == nil {
			return ScheduleTime(parsedTimestamp), nil
		}
	}
	return ScheduleTime{}, err
}

func (scheduleTime ScheduleTime) Value() (driver.Value, error) {
	return time.Time(scheduleTime), nil
}

func (scheduleTime ScheduleTime) IsZero() bool { return time.Time(scheduleTime).IsZero() }
func (scheduleTime ScheduleTime) Hour() int    { return time.Time(scheduleTime).Hour() }
func (scheduleTime ScheduleTime) Minute() int  { return time.Time(scheduleTime).Minute() }
func (scheduleTime ScheduleTime) Second() int  { return time.Time(scheduleTime).Second() }
func (scheduleTime ScheduleTime) Add(duration time.Duration) ScheduleTime {
	return ScheduleTime(time.Time(scheduleTime).Add(duration))
}
func (scheduleTime ScheduleTime) Sub(other ScheduleTime) time.Duration {
	return time.Time(scheduleTime).Sub(time.Time(other))
}

// APITimestamp is a UTC timestamp with controlled API serialization.
type APITimestamp time.Time

func parseCalendarDate(value string) (CalendarDate, error) {
	if len(value) != len(DateFormat) {
		return CalendarDate{}, errors.New("invalid calendar date format")
	}
	parsedDate, err := time.Parse(DateFormat, value)
	if err != nil {
		return CalendarDate{}, err
	}
	return CalendarDate(parsedDate), nil
}

func parseScheduleTime(value string) (ScheduleTime, error) {
	if len(value) != len(ScheduleTimeFormat) {
		return ScheduleTime{}, errors.New("invalid schedule time format")
	}
	parsedTime, err := time.Parse(ScheduleTimeFormat, value)
	if err != nil {
		return ScheduleTime{}, err
	}
	return ScheduleTime(parsedTime), nil
}

func formatScheduleTime(value ScheduleTime) ScheduleTime { return value }

func (date CalendarDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(date).Format(DateFormat))
}

func (date *CalendarDate) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsedDate, err := parseCalendarDate(value)
	if err != nil {
		return err
	}
	*date = parsedDate
	return nil
}

func (scheduleTime ScheduleTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(scheduleTime).Format(ScheduleTimeFormat))
}

func (scheduleTime *ScheduleTime) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsedTime, err := parseScheduleTime(value)
	if err != nil {
		return err
	}
	*scheduleTime = parsedTime
	return nil
}

func (timestamp APITimestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(timestamp).UTC().Format(TimestampFormat))
}

func (timestamp *APITimestamp) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value) != len(TimestampFormat) {
		return errors.New("invalid timestamp format")
	}
	parsedTimestamp, err := time.Parse(TimestampFormat, value)
	if err != nil {
		return err
	}
	*timestamp = APITimestamp(parsedTimestamp.UTC())
	return nil
}
