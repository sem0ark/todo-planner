package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPostDateEventsAutoCreatesAndDeduplicates(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "event-migration-user", "password123")
	category := createTestCategory(t, database, user.ID, "Working", "#4A90D9")
	device, err := api.deviceRepo.Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
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
		DayTemplateID *int `json:"day_template_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&first); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}
	if len(first.AcceptedEvents) != 1 || first.AcceptedEvents[0].ClientEventID != "event-1" {
		t.Fatalf("unexpected accepted events: %+v", first.AcceptedEvents)
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

func TestPostDateEventsRequiresAmendmentFields(t *testing.T) {
	event := DayEventInput{ClientEventID: "amendment-1", EventType: "amendment", OccurredAt: time.Now().UTC()}
	if err := validateDateEvents([]DayEventInput{event}); err == nil {
		t.Fatal("expected incomplete amendment to be rejected")
	}
}

func postDateEventsRequest(t *testing.T, api *API, userID int, calendarDate string, input DayEventsInput) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/days/"+calendarDate+"/events", bytes.NewReader(body))
	request = request.WithContext(withUserID(context.Background(), userID))
	response := httptest.NewRecorder()
	api.daysHandler(response, request)
	return response
}
