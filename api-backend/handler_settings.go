package main

import (
	"encoding/json"
	"net/http"
	"regexp"
)

type UserSettingsInput struct {
	DayBoundaryTime string `json:"day_boundary_time"`
}

var timeFormatRegex = regexp.MustCompile(`^([0-1][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`)

func (api *API) getSettingsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	settings, err := api.settingsRepo.GetOrCreate(r.Context(), userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to retrieve settings", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (api *API) putSettingsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input UserSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	if !timeFormatRegex.MatchString(input.DayBoundaryTime) {
		http.Error(w, "invalid time format, expected HH:MM:SS", http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
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
