package main

import (
	"encoding/json"
	"net/http"
)

type UserSettingsInput struct {
	DayBoundaryTime ScheduleTime `json:"day_boundary_time"`
}

type PublicSettings struct {
	DayBoundaryTime ScheduleTime `json:"day_boundary_time"`
	UpdatedAt       APITimestamp `json:"updated_at"`
}

func toPublicSettings(settings UserSettings) PublicSettings {
	return PublicSettings{
		DayBoundaryTime: formatScheduleTime(settings.DayBoundaryTime),
		UpdatedAt:       APITimestamp(settings.UpdatedAt),
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

	settings, err := api.settingsRepo.Update(r.Context(), userID, input.DayBoundaryTime)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update settings", err, map[string]interface{}{
			"user_id":           userID,
			"day_boundary_time": input.DayBoundaryTime,
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
