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
	var parseError error
	for _, layout := range []string{"15:04:05", "15:04", time.RFC3339} {
		parsedTime, err := time.Parse(layout, startTime)
		if err == nil {
			return parsedTime, nil
		}
		parseError = err
	}

	return time.Time{}, parseError
}
