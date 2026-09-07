package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

type publicBlockReplacement struct {
	CategoryID      *int         `json:"category_id"`
	BlockType       string       `json:"block_type"`
	StartTime       ScheduleTime `json:"start_time"`
	DurationMinutes int          `json:"duration_minutes"`
}

type publicBlockReplacementRequest struct {
	Actual *[]publicBlockReplacement `json:"actual"`
}

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
	actualBlocks := make([]ActualBlockInput, 0, len(*publicInput.Actual))
	for _, block := range *publicInput.Actual {
		actualBlocks = append(actualBlocks, ActualBlockInput{CategoryID: block.CategoryID,
			BlockType: block.BlockType, StartTime: block.StartTime,
			DurationMinutes: block.DurationMinutes})
	}
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
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	record, err := api.dayRecordRepo.FindByDate(request.Context(), userID, parsedDate)
	if err == nil {
		_, err = api.dayRecordRepo.ReplaceActualBlocks(request.Context(), record.ID, userID, actualBlocks)
		if err == nil {
			record, err = api.dayRecordRepo.FindByDate(request.Context(), userID, parsedDate)
		}
	}
	if errors.Is(err, ErrUnknownCategoryID) || errors.Is(err, ErrInvalidActualBlockType) || errors.Is(err, ErrActualBlockCategoryRequired) || errors.Is(err, ErrBlankBlockCategoryForbidden) || errors.Is(err, ErrInvalidBlockStartTime) || errors.Is(err, ErrInvalidBlockGranularity) || errors.Is(err, ErrBlockExceedsDay) || errors.Is(err, ErrActualBlocksOverlap) {
		http.Error(responseWriter, err.Error(), 400)
		return
	}
	if errors.Is(err, ErrDayRecordNotFound) {
		http.Error(responseWriter, "day record not found", 404)
		return
	}
	if err != nil {
		HTTPError(responseWriter, request, api.logger, 500, "failed to replace actual blocks", err, nil)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}
