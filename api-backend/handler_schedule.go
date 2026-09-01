package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Weekly schedule update request data
type WeeklyScheduleInput struct {
	WeeklySchedule []WeeklyScheduleEntry `json:"weekly_schedule"`
}

// Schedule override creation/update request data
type ScheduleOverrideInput struct {
	DayTemplateID *int `json:"day_template_id"`
}

// ScheduleResponse for GET /schedule
type ScheduleResponse struct {
	WeeklySchedule []WeeklySchedule   `json:"weekly_schedule"`
	Overrides      []ScheduleOverride `json:"overrides"`
}

// WeeklyScheduleResponse for PUT /schedule/weekly
type WeeklyScheduleResponse struct {
	WeeklySchedule []WeeklySchedule `json:"weekly_schedule"`
}

// TodayScheduleResponse contains the template currently assigned to today.
type TodayScheduleResponse struct {
	CalendarDate  string       `json:"calendar_date"`
	DayTemplateID *int         `json:"day_template_id"`
	Template      *DayTemplate `json:"template"`
}

// getScheduleHandler returns the full weekly schedule and all future overrides
func (api *API) getScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get weekly schedule
	weeklySchedule, err := api.scheduleRepo.GetWeeklySchedule(r.Context(), userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch weekly schedules", err, map[string]interface{}{"user_id": userID})
		return
	}

	// Get future overrides
	overrides, err := api.scheduleRepo.GetFutureOverrides(r.Context(), userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch schedule overrides", err, map[string]interface{}{"user_id": userID})
		return
	}

	response := ScheduleResponse{
		WeeklySchedule: weeklySchedule,
		Overrides:      overrides,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getTodayScheduleHandler returns today's resolved template without reading a day record.
func (api *API) getTodayScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	today := time.Now().Format("2006-01-02")
	templateID, err := api.scheduleRepo.GetTemplateForDate(r.Context(), userID, today)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to resolve today's schedule", err, map[string]interface{}{"user_id": userID, "date": today})
		return
	}

	var template *DayTemplate
	if templateID != nil {
		template, err = api.dayTemplateRepo.FindByID(r.Context(), *templateID, userID)
		if err != nil {
			HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch today's template", err, map[string]interface{}{"user_id": userID, "template_id": *templateID})
			return
		}
	}

	response := TodayScheduleResponse{
		CalendarDate:  today,
		DayTemplateID: templateID,
		Template:      template,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// putWeeklyScheduleHandler replaces all 7 day-of-week assignments
func (api *API) putWeeklyScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input WeeklyScheduleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if len(input.WeeklySchedule) != 7 {
		http.Error(w, "exactly 7 days required", http.StatusBadRequest)
		return
	}

	// Update weekly schedule
	updated, err := api.scheduleRepo.ReplaceWeeklySchedule(r.Context(), userID, input.WeeklySchedule)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "invalid day_of_week") || strings.Contains(err.Error(), "duplicate day_of_week") || strings.Contains(err.Error(), "exactly 7 days") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to save schedule", err, map[string]interface{}{"user_id": userID})
		return
	}

	response := WeeklyScheduleResponse{
		WeeklySchedule: updated,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// putScheduleOverrideHandler sets or removes override for a specific date
func (api *API) putScheduleOverrideHandler(w http.ResponseWriter, r *http.Request, dateStr string) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate date format
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Validate not in the past
	today := time.Now().Truncate(24 * time.Hour)
	if parsedDate.Before(today) {
		http.Error(w, "cannot set override for past date", http.StatusBadRequest)
		return
	}

	var input ScheduleOverrideInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Set or remove override
	override, err := api.scheduleRepo.SetOverride(r.Context(), userID, dateStr, input.DayTemplateID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to delete schedule override", err, map[string]interface{}{"user_id": userID, "date": dateStr})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(override)
}

// scheduleHandler routes to the appropriate schedule handler based on path
func (api *API) scheduleHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/schedule")

	if path == "" || path == "/" {
		// GET /schedule
		api.getScheduleHandler(w, r)
		return
	}

	if path == "/today" {
		// GET /schedule/today
		api.getTodayScheduleHandler(w, r)
		return
	}

	if path == "/weekly" {
		// PUT /schedule/weekly
		api.putWeeklyScheduleHandler(w, r)
		return
	}

	if strings.HasPrefix(path, "/overrides/") {
		// PUT /schedule/overrides/{date}
		dateStr := strings.TrimPrefix(path, "/overrides/")
		api.putScheduleOverrideHandler(w, r, dateStr)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}
