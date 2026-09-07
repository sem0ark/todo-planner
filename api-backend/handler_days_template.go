package main

import (
	"encoding/json"
	"net/http"
)

func (api *API) putDateTemplate(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	parsedDate, err := parseCalendarDate(calendarDate)
	if err != nil {
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	var input DayRecordTemplateInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(responseWriter, "invalid JSON", http.StatusBadRequest)
		return
	}
	record, err := api.dayRecordRepo.UpdateTemplateByDate(request.Context(), userID, parsedDate, input.DayTemplateID)
	if err != nil {
		if writeAppError(responseWriter, err) {
			return
		}
		HTTPError(responseWriter, request, api.logger, http.StatusInternalServerError, "failed to update day template", err, nil)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}
