package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

func (api *API) putDateTemplate(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	var input DayRecordTemplateInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(responseWriter, "invalid JSON", http.StatusBadRequest)
		return
	}
	record, err := api.dayService.UpdateTemplate(request.Context(), userID, calendarDate, input.DayTemplateID)
	if errors.Is(err, ErrDayRecordPast) {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, ErrDayRecordNotFound) || errors.Is(err, ErrDayTemplateNotFound) {
		http.Error(responseWriter, "day record or template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		HTTPError(responseWriter, request, api.logger, http.StatusInternalServerError, "failed to update day template", err, nil)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}
