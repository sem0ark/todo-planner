package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// DayRecordsResponse for GET /day-records
type DayRecordsResponse struct {
	DayRecords []DayRecord `json:"day_records"`
}

// Day record creation request data
type DayRecordInput struct {
	CalendarDate string `json:"calendar_date"` // YYYY-MM-DD
}

// Day record status update request data
type DayRecordStatusInput struct {
	ReviewStatus string `json:"review_status"` // Reviewed | Ignored
}

// Batch day event submission request data
type DayEventsInput struct {
	Events []DayEventInput `json:"events"`
}

// getDayRecordsHandler returns all day records within a date range
func (api *API) getDayRecordsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	fromDate := r.URL.Query().Get("from")
	toDate := r.URL.Query().Get("to")

	if fromDate == "" || toDate == "" {
		http.Error(w, "missing required query parameters: from and to", http.StatusBadRequest)
		return
	}

	// Validate date format (basic check)
	if !isValidDateFormat(fromDate) || !isValidDateFormat(toDate) {
		http.Error(w, "invalid date format: use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Get day records
	records, err := api.dayRecordRepo.FindByDateRange(r.Context(), userID, fromDate, toDate)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch day records", err, map[string]interface{}{"user_id": userID})
		return
	}

	response := DayRecordsResponse{
		DayRecords: records,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// postDayRecordHandler creates a new day record
func (api *API) postDayRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input DayRecordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if input.CalendarDate == "" {
		http.Error(w, "missing calendar_date", http.StatusBadRequest)
		return
	}

	if !isValidDateFormat(input.CalendarDate) {
		http.Error(w, "invalid date format: use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Create day record
	record, err := api.dayRecordRepo.Create(r.Context(), userID, input.CalendarDate)
	if err != nil {
		// Check if it's a duplicate error
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to create day record", err, map[string]interface{}{"user_id": userID, "calendar_date": input.CalendarDate})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// putDayRecordStatusHandler updates the review status of a day record
func (api *API) putDayRecordStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/day-records/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "status" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid day record ID", http.StatusBadRequest)
		return
	}

	var input DayRecordStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if input.ReviewStatus != "Reviewed" && input.ReviewStatus != "Ignored" {
		http.Error(w, "review_status must be 'Reviewed' or 'Ignored'", http.StatusBadRequest)
		return
	}

	// Update status
	record, err := api.dayRecordRepo.UpdateStatus(r.Context(), id, userID, input.ReviewStatus)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "invalid review_status") || strings.Contains(err.Error(), "cannot change status") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Check if not found
		if strings.Contains(err.Error(), "no rows") {
			http.Error(w, "day record not found", http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update day record", err, map[string]interface{}{"user_id": userID, "record_id": id})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// postDayEventsHandler appends day events and recomputes actual blocks
func (api *API) postDayEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/day-records/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "events" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid day record ID", http.StatusBadRequest)
		return
	}

	var input DayEventsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate events
	if err := validateDayEvents(input.Events); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create events and recompute actual blocks
	createdEvents, actualBlocks, err := api.dayRecordRepo.CreateEvents(r.Context(), id, userID, input.Events)
	if err != nil {
		// Check error type
		if strings.Contains(err.Error(), "cannot add events") {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if strings.Contains(err.Error(), "no rows") {
			http.Error(w, "day record not found", http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to delete day record", err, map[string]interface{}{"user_id": userID, "record_id": id})
		return
	}

	response := struct {
		CreatedEvents []DayEvent    `json:"created_events"`
		ActualBlocks  []ActualBlock `json:"actual_blocks"`
	}{
		CreatedEvents: createdEvents,
		ActualBlocks:  actualBlocks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// validateDayEvents validates a batch of day events
func validateDayEvents(events []DayEventInput) error {
	if len(events) == 0 {
		return nil
	}

	// Check event types and required fields
	for i, event := range events {
		if event.EventType != "confirmation" && event.EventType != "transition" {
			return &ValidationError{Message: "invalid event_type: must be 'confirmation' or 'transition'"}
		}

		if event.EventType == "confirmation" {
			if event.CategoryID == nil {
				return &ValidationError{Message: "category_id is required for all events"}
			}
		}

		if event.EventType == "transition" {
			if event.CategoryID == nil {
				return &ValidationError{Message: "category_id is required for all events"}
			}
		}

		if event.OccurredAt.IsZero() {
			return &ValidationError{Message: "occurred_at is required"}
		}

		// Check chronological order
		if i > 0 && event.OccurredAt.Before(events[i-1].OccurredAt) {
			return &ValidationError{Message: "events must be in chronological order"}
		}
	}

	return nil
}

// Helper function to validate date format YYYY-MM-DD
func isValidDateFormat(date string) bool {
	if len(date) != 10 {
		return false
	}
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return false
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil || year < 1900 || year > 2100 {
		return false
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return false
	}
	day, err := strconv.Atoi(parts[2])
	if err != nil || day < 1 || day > 31 {
		return false
	}
	return true
}

// dayRecordsHandler routes to the appropriate day records handler based on path and method
func (api *API) dayRecordsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/day-records")

	if path == "" || path == "/" {
		// GET /day-records or POST /day-records
		if r.Method == http.MethodGet {
			api.getDayRecordsHandler(w, r)
		} else if r.Method == http.MethodPost {
			api.postDayRecordHandler(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// PUT /day-records/{id}/status
	if strings.HasSuffix(path, "/status") {
		api.putDayRecordStatusHandler(w, r)
		return
	}

	// POST /day-records/{id}/events
	if strings.HasSuffix(path, "/events") {
		api.postDayEventsHandler(w, r)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}
