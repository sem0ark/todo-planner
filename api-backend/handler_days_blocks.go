package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

func (api *API) putDateBlocks(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	var input DayRecordBlocksInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(responseWriter, "invalid JSON", 400)
		return
	}
	record, err := api.dayService.ReplaceBlocks(request.Context(), userID, calendarDate, input.ActualBlocks)
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
