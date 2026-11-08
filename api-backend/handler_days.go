package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// PublicTimelineBlock carries a UTC instant; clients present it using the
// day's timezone_offset_minutes.
type PublicTimelineBlock struct {
	CategoryID      *int         `json:"category_id"`
	BlockType       string       `json:"block_type"`
	StartAt         APITimestamp `json:"start_at"`
	DurationMinutes int          `json:"duration_minutes"`
}

type PublicDayEvent struct {
	EventType  string       `json:"event_type"`
	CategoryID *int         `json:"category_id"`
	Text       *string      `json:"text"`
	OccurredAt APITimestamp `json:"occurred_at"`
}

type PublicDayRecord struct {
	CalendarDate          CalendarDate          `json:"calendar_date"`
	DayTemplateID         *int                  `json:"day_template_id"`
	TimezoneOffsetMinutes int                   `json:"timezone_offset_minutes"`
	Plan                  []PublicTemplateBlock `json:"plan"`
	Actual                []PublicTimelineBlock `json:"actual"`
	Events                []PublicDayEvent      `json:"events"`
	CreatedAt             APITimestamp          `json:"created_at"`
	UpdatedAt             APITimestamp          `json:"updated_at"`
}

type PublicDayRangeEntry struct {
	CalendarDate CalendarDate     `json:"calendar_date"`
	DayRecord    *PublicDayRecord `json:"day_record"`
}

type PublicDayRangeResponse struct {
	Days []PublicDayRangeEntry `json:"days"`
}

func toPublicDayRecord(record *DayRecord) PublicDayRecord {
	actualBlocks := make([]PublicTimelineBlock, 0, len(record.ActualBlocks))
	for _, block := range record.ActualBlocks {
		actualBlocks = append(actualBlocks, PublicTimelineBlock{
			CategoryID: block.CategoryID, BlockType: block.BlockType,
			StartAt: APITimestamp(block.StartAt), DurationMinutes: block.DurationMinutes,
		})
	}
	events := make([]PublicDayEvent, 0, len(record.Events))
	for _, event := range record.Events {
		events = append(events, PublicDayEvent{
			EventType: event.EventType, CategoryID: event.CategoryID, Text: event.Text,
			OccurredAt: APITimestamp(event.OccurredAt),
		})
	}
	return PublicDayRecord{
		CalendarDate: record.CalendarDate, DayTemplateID: record.DayTemplateID,
		TimezoneOffsetMinutes: record.TimezoneOffsetMinutes,
		Plan:                  toPublicTemplateBlocks(record.SnapshotBlocks),
		Actual:                actualBlocks, Events: events,
		CreatedAt: APITimestamp(record.CreatedAt), UpdatedAt: APITimestamp(record.UpdatedAt),
	}
}

func (api *API) getDays(responseWriter http.ResponseWriter, request *http.Request, userID int) {
	fromDate, fromError := parseCalendarDate(request.URL.Query().Get("from"))
	toDate, toError := parseCalendarDate(request.URL.Query().Get("to"))
	if fromError != nil || toError != nil || toDate.Before(fromDate) {
		http.Error(responseWriter, ErrInvalidDayDateRange.Error(), http.StatusBadRequest)
		return
	}
	records, err := api.dayTrackingService.FindDateRange(request.Context(), userID, fromDate, toDate)
	if err != nil {
		HTTPError(responseWriter, request, api.logger, 500, "failed to fetch days", err, nil)
		return
	}
	publicEntries := make([]PublicDayRangeEntry, 0, len(records))
	for _, entry := range records {
		var publicRecord *PublicDayRecord
		if entry.Record != nil {
			convertedRecord := toPublicDayRecord(entry.Record)
			publicRecord = &convertedRecord
		}
		publicEntries = append(publicEntries, PublicDayRangeEntry{
			CalendarDate: entry.CalendarDate,
			DayRecord:    publicRecord,
		})
	}
	writeJSON(responseWriter, PublicDayRangeResponse{Days: publicEntries})
}

func (api *API) getDay(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	parsedDate, err := parseCalendarDate(calendarDate)
	if err != nil {
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	record, err := api.dayTrackingService.FindByDate(request.Context(), userID, parsedDate)
	if err != nil {
		if writeAppError(responseWriter, err) {
			return
		}
		HTTPError(responseWriter, request, api.logger, 500, "failed to fetch day", err, nil)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}

type dayTimezoneRequest struct {
	TimezoneOffsetMinutes *int `json:"timezone_offset_minutes"`
}

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
	record, err := api.dayTrackingService.UpdateTemplateByDate(request.Context(), userID, parsedDate, input.DayTemplateID)
	if err != nil {
		if writeAppError(responseWriter, err) {
			return
		}
		HTTPError(responseWriter, request, api.logger, http.StatusInternalServerError, "failed to update day template", err, nil)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}
