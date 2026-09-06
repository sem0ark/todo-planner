package main

import (
	"encoding/json"
	"errors"
	"net/http"
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

type ScheduleOverrideDeleteResponse struct {
	CalendarDate string `json:"calendar_date"`
	Deleted      bool   `json:"deleted"`
}

// WeeklyScheduleResponse for PUT /schedule/weekly
type WeeklyScheduleResponse struct {
	WeeklySchedule []WeeklySchedule `json:"weekly_schedule"`
}

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
		if errors.Is(err, ErrInvalidWeeklySchedule) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrDayTemplateNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
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
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		http.Error(w, "invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Validate not in the past
	if dateStr < time.Now().Format("2006-01-02") {
		http.Error(w, "cannot set override for past date", http.StatusBadRequest)
		return
	}

	var input ScheduleOverrideInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.DayTemplateID == nil {
		http.Error(w, "day_template_id is required", http.StatusBadRequest)
		return
	}

	// Set the override. Removal has its own DELETE endpoint.
	override, err := api.scheduleRepo.SetOverride(r.Context(), userID, dateStr, input.DayTemplateID)
	if err != nil {
		if errors.Is(err, ErrDayTemplateNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update schedule override", err, map[string]interface{}{"user_id": userID, "date": dateStr})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(override)
}

func (api *API) deleteScheduleOverrideHandler(w http.ResponseWriter, r *http.Request, dateString string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, authenticated := getUserID(r.Context())
	if !authenticated {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := time.Parse("2006-01-02", dateString); err != nil || dateString < time.Now().Format("2006-01-02") {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}
	if err := api.scheduleRepo.DeleteOverride(r.Context(), userID, dateString); err != nil {
		if errors.Is(err, ErrScheduleOverrideNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to delete schedule override", err, nil)
		return
	}
	writeJSON(w, ScheduleOverrideDeleteResponse{CalendarDate: dateString, Deleted: true})
}

// scheduleHandler routes to the appropriate schedule handler based on path
