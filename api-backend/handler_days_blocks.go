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
	Actual              *[]publicBlockReplacement `json:"actual"`
	ClientOffsetMinutes *int                      `json:"client_offset_minutes"`
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
	record, err := api.dayRecordRepo.FindByDate(request.Context(), userID, parsedDate)
	if errors.Is(err, ErrDayRecordNotFound) {
		record, err = api.dayRecordRepo.Create(request.Context(), userID, parsedDate)
	}
	if err != nil {
		if writeAppError(responseWriter, err) {
			return
		}
		HTTPError(responseWriter, request, api.logger, 500, "failed to retrieve day record", err, nil)
		return
	}
	storageOffsetMinutes := publicInput.ClientOffsetMinutes
	if record.TimezoneOffsetMinutes != nil {
		storageOffsetMinutes = record.TimezoneOffsetMinutes
	}
	actualBlocks := make([]ActualBlockInput, 0, len(*publicInput.Actual))
	for _, block := range *publicInput.Actual {
		actualBlocks = append(actualBlocks, ActualBlockInput{CategoryID: block.CategoryID,
			BlockType: block.BlockType, StartTime: block.StartTime,
			DurationMinutes: block.DurationMinutes})
	}
	if storageOffsetMinutes != nil {
		actualBlocks = adjustActualBlocksForStorage(
			actualBlocks,
			*storageOffsetMinutes,
		)
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
	if err == nil {
		_, err = api.dayRecordRepo.ReplaceActualBlocks(request.Context(), record.ID, userID, actualBlocks, publicInput.ClientOffsetMinutes)
		if err == nil {
			record, err = api.dayRecordRepo.FindByDate(request.Context(), userID, parsedDate)
		}
	}
	if err != nil {
		if writeAppError(responseWriter, err) {
			return
		}
		HTTPError(responseWriter, request, api.logger, 500, "failed to replace actual blocks", err, nil)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}

func adjustActualBlocksForStorage(blocks []ActualBlockInput, clientOffsetMinutes int) []ActualBlockInput {
	adjustedBlocks := make([]ActualBlockInput, len(blocks))
	copy(adjustedBlocks, blocks)
	for blockIndex := range adjustedBlocks {
		adjustedBlocks[blockIndex].StartTime = shiftScheduleTime(
			adjustedBlocks[blockIndex].StartTime,
			-clientOffsetMinutes,
		)
	}
	return adjustedBlocks
}
