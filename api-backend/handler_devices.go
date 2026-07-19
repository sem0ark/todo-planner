package main

import (
	"encoding/json"
	"net/http"
)

// Device registration request data
type DeviceInput struct {
	Platform string `json:"platform"` // desktop | mobile | web
}

type DeviceResponse struct {
	DeviceID     int    `json:"device_id"`
	RegisteredAt string `json:"registered_at"`
}

func (api *API) registerDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input DeviceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Platform == "" {
		http.Error(w, "platform is required", http.StatusBadRequest)
		return
	}

	if input.Platform != "desktop" && input.Platform != "mobile" && input.Platform != "web" {
		http.Error(w, "invalid platform value", http.StatusBadRequest)
		return
	}

	device, err := api.deviceRepo.Create(r.Context(), userID, input.Platform)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := DeviceResponse{
		DeviceID:     device.ID,
		RegisteredAt: device.RegisteredAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
