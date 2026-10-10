package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestPutDateTemplate(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "put-day-template", "password123")
	template := createTestDayTemplate(t, database, user.ID, "Requested", nil)
	calendarDate := calendarDateRelativeToToday(t, database, 1)
	if _, err := api.dayTrackingService.CreateDayRecord(context.Background(), user.ID, calendarDate); err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	requestBody := `{"day_template_id":` + strconv.Itoa(template.ID) + `}`
	request := httptest.NewRequest(
		http.MethodPut,
		"/days/"+calendarDate.String()+"/template",
		strings.NewReader(requestBody),
	)
	response := httptest.NewRecorder()

	api.putDateTemplate(response, request, user.ID, calendarDate.String())

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload PublicDayRecord
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode day-template response: %v", err)
	}
	if payload.DayTemplateID == nil || *payload.DayTemplateID != template.ID {
		t.Fatalf("expected template ID %d, got %v", template.ID, payload.DayTemplateID)
	}
}

func TestPutDateTemplateRejectsMalformedRequests(t *testing.T) {
	database := setupTestDB(t)
	api := NewAPI(database, "test-secret", NewLogger("test"))
	user := createTestUser(t, database, "invalid-put-day-template", "password123")
	calendarDate := calendarDateRelativeToToday(t, database, 1)
	if _, err := api.dayTrackingService.CreateDayRecord(context.Background(), user.ID, calendarDate); err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	testCases := []struct {
		name         string
		calendarDate string
		requestBody  string
		expectedCode int
	}{
		{
			name:         "invalid date",
			calendarDate: "not-a-date",
			requestBody:  `{}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid JSON",
			calendarDate: calendarDate.String(),
			requestBody:  `{"day_template_id":`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "missing template",
			calendarDate: calendarDate.String(),
			requestBody:  `{"day_template_id":999999}`,
			expectedCode: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPut,
				"/days/"+testCase.calendarDate+"/template",
				strings.NewReader(testCase.requestBody),
			)
			response := httptest.NewRecorder()

			api.putDateTemplate(response, request, user.ID, testCase.calendarDate)

			if response.Code != testCase.expectedCode {
				t.Fatalf("expected %d, got %d: %s", testCase.expectedCode, response.Code, response.Body.String())
			}
		})
	}
}
