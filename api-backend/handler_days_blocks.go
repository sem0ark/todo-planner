package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

type publicBlockReplacement struct {
	CategoryID      *int            `json:"category_id"`
	BlockType       string          `json:"block_type"`
	StartTime       APIScheduleTime `json:"start_time"`
	DurationMinutes int             `json:"duration_minutes"`
}

type publicBlockReplacementRequest struct {
	Actual *[]publicBlockReplacement `json:"actual"`
}

func (api *API) putDateBlocks(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
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
			BlockType: block.BlockType, StartTime: string(block.StartTime),
			DurationMinutes: block.DurationMinutes})
	}
	record, err := api.dayService.ReplaceBlocks(request.Context(), userID, calendarDate, actualBlocks)
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
