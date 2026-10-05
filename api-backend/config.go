package main

import "time"

const (
	// minimumActualBlockDuration is the shortest event-derived block retained
	// during actual timeline normalization.
	minimumActualBlockDuration = 5 * time.Minute

	// initPlanDays is the number of consecutive days returned by the bootstrap
	// endpoint, including the requested calendar date.
	initPlanDays = 7
)
