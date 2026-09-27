package main

import (
	"encoding/json"
	"net/http"
)

type UserSettingsInput struct {
	DayRangeStartTime ScheduleTime `json:"day_range_start_time"`
	DayRangeEndTime   ScheduleTime `json:"day_range_end_time"`
}

// Tracking day range settings are rendering hints only. They do not affect
// event calendar-date assignment or actual block derivation.

type PublicSettings struct {
	DayRangeStartTime ScheduleTime `json:"day_range_start_time"`
	DayRangeEndTime   ScheduleTime `json:"day_range_end_time"`
	UpdatedAt         APITimestamp `json:"updated_at"`
}

func toPublicSettings(settings UserSettings) PublicSettings {
	return PublicSettings{
		DayRangeStartTime: formatScheduleTime(settings.DayRangeStartTime),
		DayRangeEndTime:   formatScheduleTime(settings.DayRangeEndTime),
		UpdatedAt:         APITimestamp(settings.UpdatedAt),
	}
}

func (api *API) getSettingsHandler(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromRequest(r)

	settings, err := api.settingsRepo.Get(r.Context(), userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to retrieve settings", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	writeJSON(w, toPublicSettings(*settings))
}

func (api *API) putSettingsHandler(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromRequest(r)

	var input UserSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	settings, err := api.settingsRepo.Update(r.Context(), userID, input.DayRangeStartTime, input.DayRangeEndTime)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update settings", err, map[string]interface{}{
			"user_id":              userID,
			"day_range_start_time": input.DayRangeStartTime,
			"day_range_end_time":   input.DayRangeEndTime,
		})
		return
	}

	writeJSON(w, toPublicSettings(*settings))
}

func (api *API) settingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.getSettingsHandler(w, r)
	case http.MethodPut:
		api.putSettingsHandler(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
