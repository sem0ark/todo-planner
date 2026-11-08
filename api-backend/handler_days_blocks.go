package main

import (
	"encoding/json"
	"net/http"
)

type publicBlockReplacement struct {
	CategoryID      *int         `json:"category_id"`
	BlockType       string       `json:"block_type"`
	StartAt         APITimestamp `json:"start_at"`
	DurationMinutes int          `json:"duration_minutes"`
}

type publicBlockReplacementRequest struct {
	Actual *[]publicBlockReplacement `json:"actual"`
}

// putDateBlocks replaces the day's actual blocks during review. The day must exist.
func (api *API) putDateBlocks(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	parsedDate, err := parseCalendarDate(calendarDate)
	if err != nil {
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	var publicInput publicBlockReplacementRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&publicInput); err != nil {
		http.Error(responseWriter, "invalid JSON", 400)
		return
	}
	if publicInput.Actual == nil {
		http.Error(responseWriter, "actual is required", http.StatusBadRequest)
		return
	}
	record, err := api.dayRecordRepo.FindByDate(request.Context(), userID, parsedDate)
	if err != nil {
		api.writeDayError(responseWriter, request, "failed to retrieve day record", err)
		return
	}
	actualBlocks := make([]ActualBlockInput, 0, len(*publicInput.Actual))
	for _, block := range *publicInput.Actual {
		actualBlocks = append(actualBlocks, ActualBlockInput{
			CategoryID: block.CategoryID, BlockType: block.BlockType,
			StartAt: block.StartAt.Time(), DurationMinutes: block.DurationMinutes,
		})
	}
	actualBlocks = resolveOverlappingActualBlocks(actualBlocks)
	if err := validateActualBlocks(actualBlocks); err != nil {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	categoryIDs := make([]int, 0, len(actualBlocks))
	for _, block := range actualBlocks {
		if block.CategoryID != nil {
			categoryIDs = append(categoryIDs, *block.CategoryID)
		}
	}
	if err := api.categoryRepo.ValidateIDs(request.Context(), userID, categoryIDs); err != nil {
		if !writeAppError(responseWriter, err) {
			http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		}
		return
	}
	if _, err := api.dayTrackingService.ReplaceActualBlocks(request.Context(), record.ID, userID, actualBlocks); err != nil {
		api.writeDayError(responseWriter, request, "failed to replace actual blocks", err)
		return
	}
	record, err = api.dayTrackingService.FindByDate(request.Context(), userID, parsedDate)
	if err != nil {
		api.writeDayError(responseWriter, request, "failed to replace actual blocks", err)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}
