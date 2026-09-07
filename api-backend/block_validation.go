package main

import "time"

const minutesPerDay = 24 * 60

func blockExceedsDay(startTime string, durationMinutes int) bool {
	parsedTime, err := parseBlockStartTime(startTime)
	if err != nil {
		return false
	}

	startMinute := parsedTime.Hour()*60 + parsedTime.Minute()
	return startMinute+durationMinutes > minutesPerDay
}

func parseBlockStartTime(startTime string) (time.Time, error) {
	return time.Parse(ScheduleTimeFormat, startTime)
}
