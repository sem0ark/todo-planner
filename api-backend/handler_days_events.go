package main

import (
	"encoding/json"
	"errors"
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
	var publicInput dayEventsRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&publicInput); err != nil {
		http.Error(responseWriter, "invalid JSON", 400)
		return
	}
	input := toInternalDayEventsRequest(publicInput)
	result, err := api.dayService.AppendEvents(request.Context(), userID, calendarDate, input)
	if errors.Is(err, ErrDeviceIDRequired) || errors.Is(err, ErrUnknownCategoryID) || errors.Is(err, ErrMissingEventCategory) || errors.Is(err, ErrInvalidEventType) || errors.Is(err, ErrIncompleteAmendment) || errors.Is(err, ErrMissingEventTimestamp) || errors.Is(err, ErrUnsortedEvents) || errors.Is(err, ErrMissingClientEventID) {
		http.Error(responseWriter, err.Error(), 400)
		return
	}
	if errors.Is(err, ErrDeviceNotFound) {
		http.Error(responseWriter, err.Error(), 404)
		return
	}
	if errors.Is(err, ErrAmendmentTargetNotFound) {
		http.Error(responseWriter, err.Error(), 400)
		return
	}
	if errors.Is(err, ErrNonMonotonicTransitions) {
		http.Error(responseWriter, err.Error(), 409)
		return
	}
	if err != nil {
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
