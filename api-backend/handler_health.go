package main

import (
	"context"
	"net/http"
	"time"
)

type healthResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (api *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := api.db.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, healthResponse{Status: "unhealthy", Error: err.Error()})
		return
	}

	writeJSON(w, healthResponse{Status: "healthy"})
}
