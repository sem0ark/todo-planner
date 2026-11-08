package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type dayTimezoneRequest struct {
	TimezoneOffsetMinutes *int `json:"timezone_offset_minutes"`
}

// decodeDayTimezoneRequest reads the optional offset body of POST/PUT /days/{date}.
func decodeDayTimezoneRequest(request *http.Request) (dayTimezoneRequest, error) {
	var input dayTimezoneRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		return input, NewBadRequestError("invalid JSON")
	}
	if input.TimezoneOffsetMinutes != nil {
		if err := validateTimezoneOffsetMinutes(*input.TimezoneOffsetMinutes); err != nil {
			return input, err
		}
	}
	return input, nil
}

func (api *API) writeDayError(responseWriter http.ResponseWriter, request *http.Request, message string, err error) {
	if writeAppError(responseWriter, err) {
		return
	}
	HTTPError(responseWriter, request, api.logger, http.StatusInternalServerError, message, err, nil)
}

// createDay creates the day record. The offset defaults to 0 (UTC) when omitted.
func (api *API) createDay(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	parsedDate, err := parseCalendarDate(calendarDate)
	if err != nil {
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	input, err := decodeDayTimezoneRequest(request)
	if err != nil {
		writeAppError(responseWriter, err)
		return
	}
	offsetMinutes := 0
	if input.TimezoneOffsetMinutes != nil {
		offsetMinutes = *input.TimezoneOffsetMinutes
	}
	record, err := api.dayTrackingService.CreateDayRecord(request.Context(), userID, parsedDate, offsetMinutes)
	if err != nil {
		api.writeDayError(responseWriter, request, "failed to create day", err)
		return
	}
	responseWriter.WriteHeader(http.StatusCreated)
	writeJSON(responseWriter, toPublicDayRecord(record))
}

// updateDay overrides the day's timezone offset. Events and blocks stay in UTC.
func (api *API) updateDay(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	parsedDate, err := parseCalendarDate(calendarDate)
	if err != nil {
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	input, err := decodeDayTimezoneRequest(request)
	if err != nil {
		writeAppError(responseWriter, err)
		return
	}
	if input.TimezoneOffsetMinutes == nil {
		writeAppError(responseWriter, ErrInvalidTimezoneOffset)
		return
	}
	record, err := api.dayTrackingService.UpdateTimezoneOffset(request.Context(), userID, parsedDate, *input.TimezoneOffsetMinutes)
	if err != nil {
		api.writeDayError(responseWriter, request, "failed to update day", err)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}
