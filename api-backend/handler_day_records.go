package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DayRecordsResponse for GET /day-records
type DayRecordsResponse struct {
	DayRecords []DayRecord `json:"day_records"`
}

const maximumDayRecordRangeDays = 31

// Day record creation request data
type DayRecordInput struct {
	CalendarDate string `json:"calendar_date"` // YYYY-MM-DD
}

// Batch day event submission request data
type DayEventsInput struct {
	Events []DayEventInput `json:"events"`
}

type DayRecordBlocksInput struct {
	ActualBlocks []ActualBlockInput `json:"actual_blocks"`
}

type DayRecordTemplateInput struct {
	DayTemplateID *int `json:"day_template_id"`
}

type ActualBlockInput struct {
	CategoryID      *int   `json:"category_id"`
	BlockType       string `json:"block_type"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
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
	if !isValidCalendarDate(fromDate) || !isValidCalendarDate(toDate) {
		http.Error(w, "invalid date format: use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	fromCalendarDate, _ := time.Parse("2006-01-02", fromDate)
	toCalendarDate, _ := time.Parse("2006-01-02", toDate)
	if toCalendarDate.Before(fromCalendarDate) {
		http.Error(w, "invalid date range", http.StatusBadRequest)
		return
	}
	if toCalendarDate.Sub(fromCalendarDate) > maximumDayRecordRangeDays*24*time.Hour {
		http.Error(w, "date range cannot exceed 31 days", http.StatusBadRequest)
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

	if !isValidCalendarDate(input.CalendarDate) {
		http.Error(w, "invalid calendar date: use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	record, err := api.dayRecordRepo.Create(r.Context(), userID, input.CalendarDate)
	if err != nil {
		if errors.Is(err, ErrDayRecordAlreadyExists) {
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

	recordID, err := strconv.Atoi(parts[0])
	if err != nil || recordID <= 0 {
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
	if err := api.validateCategoryIDs(r.Context(), userID, input.Events); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdEvents, actualBlocks, err := api.dayRecordRepo.CreateEvents(r.Context(), recordID, userID, input.Events)
	if err != nil {
		if errors.Is(err, ErrDayRecordNotFound) {
			http.Error(w, "day record not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrDayRecordEventsForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to append day events", err, map[string]interface{}{"user_id": userID, "record_id": recordID})
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

// putDayRecordHandler replaces the actual blocks for a day record.
func (api *API) putDayRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	dayRecordID, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/day-records/"))
	if err != nil || dayRecordID <= 0 {
		http.Error(w, "invalid day record ID", http.StatusBadRequest)
		return
	}

	var input DayRecordBlocksInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validateActualBlocks(input.ActualBlocks); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := api.validateActualBlockCategoryIDs(r.Context(), userID, input.ActualBlocks); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record, err := api.dayRecordRepo.FindByID(r.Context(), dayRecordID, userID)
	if err != nil {
		if errors.Is(err, ErrDayRecordNotFound) {
			http.Error(w, "day record not found", http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to load day record", err, map[string]interface{}{"user_id": userID, "day_record_id": dayRecordID})
		return
	}
	blocks, err := api.dayRecordRepo.ReplaceActualBlocks(r.Context(), record.ID, userID, input.ActualBlocks)
	if err != nil {
		if errors.Is(err, ErrDayRecordEditForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to replace actual blocks", err, map[string]interface{}{"user_id": userID, "day_record_id": dayRecordID})
		return
	}

	updatedRecord, err := api.dayRecordRepo.FindByID(r.Context(), record.ID, userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to load updated day record", err, map[string]interface{}{"user_id": userID, "day_record_id": dayRecordID})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ActualBlocks []ActualBlock `json:"actual_blocks"`
		UpdatedAt    time.Time     `json:"updated_at"`
	}{ActualBlocks: blocks, UpdatedAt: updatedRecord.UpdatedAt})
}

func (api *API) validateCategoryIDs(contextValue context.Context, userID int, events []DayEventInput) error {
	for _, event := range events {
		if event.CategoryID == nil {
			return &ValidationError{Message: "category_id is required for all events"}
		}
		category, err := api.categoryRepo.FindByID(contextValue, *event.CategoryID, userID)
		if err != nil || category.IsDeleted {
			return &ValidationError{Message: "unknown category_id"}
		}
	}
	return nil
}

func (api *API) validateActualBlockCategoryIDs(contextValue context.Context, userID int, blocks []ActualBlockInput) error {
	for _, block := range blocks {
		if block.CategoryID == nil {
			continue
		}
		category, err := api.categoryRepo.FindByID(contextValue, *block.CategoryID, userID)
		if err != nil || category.IsDeleted {
			return &ValidationError{Message: "unknown category_id"}
		}
	}
	return nil
}

func (api *API) putDayRecordTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/day-records/"), "/")
	if len(pathParts) != 2 || pathParts[1] != "template" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	recordID, err := strconv.Atoi(pathParts[0])
	if err != nil || recordID <= 0 {
		http.Error(w, "invalid day record ID", http.StatusBadRequest)
		return
	}
	var input DayRecordTemplateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	record, err := api.dayRecordRepo.UpdateTemplate(r.Context(), userID, recordID, input.DayTemplateID)
	if err != nil {
		if errors.Is(err, ErrDayRecordPast) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrDayRecordNotFound) || errors.Is(err, ErrDayTemplateNotFound) {
			http.Error(w, "day record or template not found", http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update day record template", err, map[string]interface{}{"user_id": userID, "day_record_id": recordID})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
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

func validateActualBlocks(blocks []ActualBlockInput) error {
	var previousEnd int
	for index, block := range blocks {
		if block.BlockType != "actual" && block.BlockType != "blank" {
			return &ValidationError{Message: "block_type must be 'actual' or 'blank'"}
		}
		if block.BlockType == "actual" && block.CategoryID == nil {
			return &ValidationError{Message: "category_id is required for actual blocks"}
		}
		if block.BlockType == "blank" && block.CategoryID != nil {
			return &ValidationError{Message: "category_id must be null for blank blocks"}
		}

		parsedTime, err := time.Parse("15:04:05", block.StartTime)
		if err != nil {
			parsedTime, err = time.Parse(time.RFC3339, block.StartTime)
		}
		if err != nil {
			return &ValidationError{Message: "start_time must be a valid time"}
		}
		minuteOfDay := parsedTime.Hour()*60 + parsedTime.Minute()
		if minuteOfDay%15 != 0 || parsedTime.Second() != 0 {
			return &ValidationError{Message: "start_time must use 15-minute increments"}
		}
		if block.DurationMinutes < 30 || block.DurationMinutes%15 != 0 {
			return &ValidationError{Message: "duration_minutes must be at least 30 and a multiple of 15"}
		}
		if blockExceedsDay(block.StartTime, block.DurationMinutes) {
			return &ValidationError{Message: "block must end by 24:00"}
		}
		if index > 0 && minuteOfDay < previousEnd {
			return &ValidationError{Message: "actual blocks must not overlap"}
		}
		previousEnd = minuteOfDay + block.DurationMinutes
	}
	return nil
}

func isValidCalendarDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

// dayRecordsHandler routes to the appropriate day records handler based on path and method
func (api *API) dayRecordsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/day-records")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			api.getDayRecordsHandler(w, r)
		case http.MethodPost:
			api.postDayRecordHandler(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasSuffix(path, "/template") {
		api.putDayRecordTemplateHandler(w, r)
		return
	}

	// POST /day-records/{id}/events
	if strings.HasSuffix(path, "/events") {
		api.postDayEventsHandler(w, r)
		return
	}

	// PUT /day-records/{id}
	if strings.Count(strings.Trim(path, "/"), "/") == 0 {
		api.putDayRecordHandler(w, r)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}
