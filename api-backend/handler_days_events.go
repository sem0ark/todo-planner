package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type publicAcceptedEvent struct {
	ClientEventID string       `json:"client_event_id"`
	EventType     string       `json:"event_type"`
	CategoryID    *int         `json:"category_id"`
	OccurredAt    APITimestamp `json:"occurred_at"`
}

type publicDayEventsResponse struct {
	PublicDayRecord
	AcceptedEvents    []publicAcceptedEvent `json:"accepted_events"`
	DuplicateEventIDs []string              `json:"duplicate_client_event_ids"`
}

type dayEventRequest struct {
	ClientEventID       string        `json:"client_event_id"`
	EventType           string        `json:"event_type"`
	CategoryID          *int          `json:"category_id"`
	OccurredAt          APITimestamp  `json:"occurred_at"`
	TargetClientEventID *string       `json:"target_client_event_id"`
	CorrectedAt         *APITimestamp `json:"corrected_at"`
}

type dayEventsRequest struct {
	DeviceID int               `json:"device_id"`
	Events   []dayEventRequest `json:"events"`
}

func toInternalDayEventsRequest(request dayEventsRequest) DayEventsInput {
	events := make([]DayEventInput, 0, len(request.Events))
	for _, event := range request.Events {
		targetClientEventID := ""
		if event.TargetClientEventID != nil {
			targetClientEventID = *event.TargetClientEventID
		}
		var correctedAt *time.Time
		if event.CorrectedAt != nil {
			correctedTime := time.Time(*event.CorrectedAt)
			correctedAt = &correctedTime
		}
		events = append(events, DayEventInput{
			ClientEventID: event.ClientEventID, EventType: event.EventType,
			CategoryID: event.CategoryID, OccurredAt: time.Time(event.OccurredAt),
			TargetClientEventID: targetClientEventID, CorrectedAt: correctedAt,
		})
	}
	return DayEventsInput{DeviceID: request.DeviceID, Events: events}
}

func (api *API) postDateEvents(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	parsedDate, err := parseCalendarDate(calendarDate)
	if err != nil {
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	var publicInput dayEventsRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&publicInput); err != nil {
		http.Error(responseWriter, "invalid JSON", 400)
		return
	}
	input := toInternalDayEventsRequest(publicInput)
	if input.DeviceID <= 0 {
		http.Error(responseWriter, ErrDeviceIDRequired.Error(), http.StatusBadRequest)
		return
	}
	if err := validateDateEvents(input.Events); err != nil {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	categoryIDs := make([]int, 0, len(input.Events))
	for _, event := range input.Events {
		if event.CategoryID != nil {
			categoryIDs = append(categoryIDs, *event.CategoryID)
		}
	}
	if err := api.categoryRepo.ValidateIDs(request.Context(), userID, categoryIDs); err != nil {
		if !writeAppError(responseWriter, err) {
			http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		}
		return
	}
	result, err := api.dayRecordRepo.CreateEventsByDate(request.Context(), userID, parsedDate, input.DeviceID, input.Events)
	if err != nil {
		if !writeAppError(responseWriter, err) {
			return
		}
		HTTPError(responseWriter, request, api.logger, 500, "failed to append day events", err, nil)
		return
	}
	acceptedEvents := make([]publicAcceptedEvent, 0, len(result.AcceptedEvents))
	for _, event := range result.AcceptedEvents {
		clientEventID := ""
		if event.ClientEventID != nil {
			clientEventID = *event.ClientEventID
		}
		acceptedEvents = append(acceptedEvents, publicAcceptedEvent{clientEventID, event.EventType, event.CategoryID, APITimestamp(event.OccurredAt)})
	}
	writeJSON(responseWriter, publicDayEventsResponse{toPublicDayRecord(result.Record), acceptedEvents, result.DuplicateEventIDs})
}
