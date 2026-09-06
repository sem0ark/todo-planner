package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type publicAcceptedEvent struct {
	ClientEventID string    `json:"client_event_id"`
	EventType     string    `json:"event_type"`
	CategoryID    *int      `json:"category_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type publicDayEventsResponse struct {
	publicDayRecord
	AcceptedEvents    []publicAcceptedEvent `json:"accepted_events"`
	DuplicateEventIDs []string              `json:"duplicate_client_event_ids"`
}

func (api *API) postDateEvents(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	var input DayEventsInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(responseWriter, "invalid JSON", 400)
		return
	}
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
		acceptedEvents = append(acceptedEvents, publicAcceptedEvent{clientEventID, event.EventType, event.CategoryID, event.OccurredAt})
	}
	writeJSON(responseWriter, publicDayEventsResponse{toPublicDayRecord(result.Record), acceptedEvents, result.DuplicateEventIDs})
}
