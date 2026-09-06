package main

import (
	"errors"
	"net/http"
)

func (api *API) createDay(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	record, err := api.dayService.CreateDay(request.Context(), userID, calendarDate)
	if errors.Is(err, ErrDayRecordAlreadyExists) {
		http.Error(responseWriter, "day record already exists", http.StatusConflict)
		return
	}
	if err != nil {
		HTTPError(responseWriter, request, api.logger, http.StatusInternalServerError, "failed to create day", err, nil)
		return
	}
	responseWriter.WriteHeader(http.StatusCreated)
	writeJSON(responseWriter, toPublicDayRecord(record))
}
