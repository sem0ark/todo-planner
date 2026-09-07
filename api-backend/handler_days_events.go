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
	startedAt := time.Now()
	api.logger.Info("Day events request started", map[string]interface{}{
		"method":        request.Method,
		"path":          request.URL.Path,
		"user_id":       userID,
		"calendar_date": calendarDate,
	})
	defer func() {
		api.logger.Info("Day events request completed", map[string]interface{}{
			"calendar_date": calendarDate,
			"duration_ms":   time.Since(startedAt).Milliseconds(),
		})
	}()

	parsedDate, err := parseCalendarDate(calendarDate)
	if err != nil {
		api.logger.Error("Invalid calendar date", err, map[string]interface{}{
			"calendar_date": calendarDate,
		})
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	api.logger.Info("Calendar date parsed", map[string]interface{}{"calendar_date": calendarDate})

	var publicInput dayEventsRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&publicInput); err != nil {
		api.logger.Error("Day events request body decode failed", err, nil)
		http.Error(responseWriter, "invalid JSON", 400)
		return
	}
	api.logger.Info("Day events request body decoded", map[string]interface{}{
		"device_id":   publicInput.DeviceID,
		"event_count": len(publicInput.Events),
	})

	input := toInternalDayEventsRequest(publicInput)
	if input.DeviceID <= 0 {
		api.logger.Error("Day events request validation failed", ErrDeviceIDRequired, map[string]interface{}{
			"device_id": input.DeviceID,
		})
		http.Error(responseWriter, ErrDeviceIDRequired.Error(), http.StatusBadRequest)
		return
	}
	if err := validateDateEvents(input.Events); err != nil {
		api.logger.Error("Day events request validation failed", err, map[string]interface{}{
			"device_id":   input.DeviceID,
			"event_count": len(input.Events),
		})
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	api.logger.Info("Day events request validation passed", map[string]interface{}{
		"device_id":   input.DeviceID,
		"event_count": len(input.Events),
	})

	categoryIDs := make([]int, 0, len(input.Events))
	for _, event := range input.Events {
		if event.CategoryID != nil {
			categoryIDs = append(categoryIDs, *event.CategoryID)
		}
	}
	api.logger.Info("Validating day event category IDs", map[string]interface{}{
		"category_ids": categoryIDs,
	})
	if err := api.categoryRepo.ValidateIDs(request.Context(), userID, categoryIDs); err != nil {
		api.logger.Error("Day event category validation failed", err, map[string]interface{}{
			"user_id":       userID,
			"device_id":     input.DeviceID,
			"category_ids":  categoryIDs,
			"calendar_date": calendarDate,
		})
		if !writeAppError(responseWriter, err) {
			http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		}
		return
	}
	api.logger.Info("Day event category IDs validated", map[string]interface{}{
		"category_ids": categoryIDs,
	})

	api.logger.Info("Creating day events", map[string]interface{}{
		"user_id":       userID,
		"calendar_date": calendarDate,
		"device_id":     input.DeviceID,
		"event_count":   len(input.Events),
	})
	result, err := api.dayRecordRepo.CreateEventsByDate(request.Context(), userID, parsedDate, input.DeviceID, input.Events)
	if err != nil {
		if writeAppError(responseWriter, err) {
			api.logger.Error("Returning day events application error", err, nil)
			return
		}
		api.logger.Error("Failed to create day events", err, nil)
		HTTPError(responseWriter, request, api.logger, 500, "failed to append day events", err, nil)
		return
	}
	api.logger.Info("Day events created", map[string]interface{}{
		"calendar_date":    calendarDate,
		"accepted_events":  len(result.AcceptedEvents),
		"duplicate_events": len(result.DuplicateEventIDs),
	})
	acceptedEvents := make([]publicAcceptedEvent, 0, len(result.AcceptedEvents))
	for _, event := range result.AcceptedEvents {
		clientEventID := ""
		if event.ClientEventID != nil {
			clientEventID = *event.ClientEventID
		}
		acceptedEvents = append(acceptedEvents, publicAcceptedEvent{clientEventID, event.EventType, event.CategoryID, APITimestamp(event.OccurredAt)})
	}
	api.logger.Info("Building day events response", map[string]interface{}{
		"accepted_events":  len(acceptedEvents),
		"duplicate_events": len(result.DuplicateEventIDs),
		"plan_blocks":      len(result.Record.SnapshotBlocks),
		"actual_blocks":    len(result.Record.ActualBlocks),
	})
	response := publicDayEventsResponse{
		PublicDayRecord:   toPublicDayRecord(result.Record),
		AcceptedEvents:    acceptedEvents,
		DuplicateEventIDs: result.DuplicateEventIDs,
	}
	writeJSON(responseWriter, response)
	api.logger.Info("Day events response written", map[string]interface{}{"calendar_date": calendarDate})
}
