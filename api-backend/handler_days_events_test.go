package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPostDateEventsDeduplicates(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "event-migration-user", "password123")
	category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	if _, err := api.dayTrackingService.CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-09-06"), 0); err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	eventTime := time.Date(2026, 9, 6, 14, 26, 37, 0, time.UTC)
	requestBody := DayEventsInput{DeviceID: device.ID, Events: []DayEventInput{{ClientEventID: "event-1", EventType: "transition", CategoryID: &category.ID, OccurredAt: eventTime}}}
	response := postDateEventsRequest(t, api, user.ID, "2026-09-06", requestBody)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var first struct {
		AcceptedEvents []struct {
			ClientEventID string `json:"client_event_id"`
		} `json:"accepted_events"`
		Events []struct {
			EventType string `json:"event_type"`
		} `json:"events"`
		DayTemplateID *int `json:"day_template_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&first); err != nil {
		t.Fatalf("failed to decode first response: %v; body=%s", err, response.Body.String())
	}
	if len(first.AcceptedEvents) != 1 || first.AcceptedEvents[0].ClientEventID != "event-1" {
		t.Fatalf("unexpected accepted events: %+v", first.AcceptedEvents)
	}
	if len(first.Events) != 1 || first.Events[0].EventType != "transition" {
		t.Fatalf("expected the response record to include the inserted event, got %+v", first.Events)
	}

	response = postDateEventsRequest(t, api, user.ID, "2026-09-06", requestBody)
	if response.Code != http.StatusOK {
		t.Fatalf("expected retry 200, got %d", response.Code)
	}
	var retry struct {
		AcceptedEvents []DayEvent `json:"accepted_events"`
		DuplicateIDs   []string   `json:"duplicate_client_event_ids"`
	}
	if err := json.NewDecoder(response.Body).Decode(&retry); err != nil {
		t.Fatalf("failed to decode retry response: %v", err)
	}
	if len(retry.AcceptedEvents) != 0 || len(retry.DuplicateIDs) != 1 || retry.DuplicateIDs[0] != "event-1" {
		t.Fatalf("unexpected retry result: %+v", retry)
	}
}

func TestPostDateEventsRejectsForeignDevice(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	owner := createTestUser(t, database, "event-owner", "password123")
	otherUser := createTestUser(t, database, "event-foreign", "password123")
	device, err := api.deviceRepo.Create(context.Background(), owner.ID, "mobile")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	requestBody := DayEventsInput{DeviceID: device.ID, Events: []DayEventInput{}}
	response := postDateEventsRequest(t, api, otherUser.ID, "2026-09-06", requestBody)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestPostDateEventsRequiresExistingDay(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "event-missing-day-user", "password123")
	category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	requestBody := DayEventsInput{
		DeviceID: device.ID,
		Events: []DayEventInput{{
			ClientEventID: "event-no-day",
			EventType:     "transition",
			CategoryID:    &category.ID,
			OccurredAt:    parseTime("2026-09-06T23:30:00Z"),
		}},
	}
	response := postDateEventsRequest(t, api, user.ID, "2026-09-07", requestBody)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	if _, err := api.dayRecordRepo.FindByDate(context.Background(), user.ID, mustCalendarDate("2026-09-07")); err == nil {
		t.Fatal("expected no day record to be auto-created")
	}
}

func TestPostDateEventsRejectsRemovedLocalFields(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "event-local-field-user", "password123")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	if _, err := api.dayTrackingService.CreateDayRecord(context.Background(), user.ID, mustCalendarDate("2026-09-06"), 0); err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	body := `{"device_id":` + strconv.Itoa(device.ID) + `,"events":[{"client_event_id":"e1","event_type":"note","text":"x","occurred_at":"2026-09-06T09:00:00Z","occurred_at_local":"2026-09-06T11:00:00+02:00"}]}`
	request := httptest.NewRequest(http.MethodPost, "/days/2026-09-06/events", strings.NewReader(body))
	request = request.WithContext(withUserID(context.Background(), user.ID))
	response := httptest.NewRecorder()

	api.daysHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for removed field, got %d: %s", response.Code, response.Body.String())
	}
}

func TestPostDateEventsRequiresAmendmentFields(t *testing.T) {
	event := DayEventInput{ClientEventID: "amendment-1", EventType: "amendment", OccurredAt: time.Now().UTC()}
	if err := validateDateEvents([]DayEventInput{event}); err == nil {
		t.Fatal("expected incomplete amendment to be rejected")
	}
}

func TestDayEventRequestAllowsNullableAmendmentFieldsToBeOmitted(t *testing.T) {
	var request dayEventRequest
	err := json.Unmarshal([]byte(`{
		"client_event_id": "event-1",
		"event_type": "confirmation",
		"occurred_at": "2026-09-06T14:26:37Z"
	}`), &request)
	if err != nil {
		t.Fatalf("expected request to decode: %v", err)
	}
	if request.TargetClientEventID != nil || request.CorrectedAt != nil {
		t.Fatal("expected omitted amendment fields to remain nil")
	}
}

func postDateEventsRequest(t *testing.T, api *API, userID int, calendarDate string, input DayEventsInput) *httptest.ResponseRecorder {
	t.Helper()
	publicEvents := make([]dayEventRequest, 0, len(input.Events))
	for _, event := range input.Events {
		var correctedAt *APITimestamp
		if event.CorrectedAt != nil {
			convertedTime := APITimestamp(*event.CorrectedAt)
			correctedAt = &convertedTime
		}
		var targetClientEventID *string
		if event.TargetClientEventID != "" {
			targetClientEventID = &event.TargetClientEventID
		}
		publicEvents = append(publicEvents, dayEventRequest{
			ClientEventID:       event.ClientEventID,
			EventType:           event.EventType,
			CategoryID:          event.CategoryID,
			OccurredAt:          APITimestamp(event.OccurredAt),
			TargetClientEventID: targetClientEventID,
			CorrectedAt:         correctedAt,
		})
	}
	requestBody := dayEventsRequest{DeviceID: input.DeviceID, Events: publicEvents}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/days/"+calendarDate+"/events", bytes.NewReader(body))
	request = request.WithContext(withUserID(context.Background(), userID))
	response := httptest.NewRecorder()
	api.daysHandler(response, request)
	return response
}
