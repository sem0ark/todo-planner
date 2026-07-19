package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// Sync request
type SyncRequest struct {
	DeviceID   int              `json:"device_id"`
	LastSyncAt *time.Time       `json:"last_sync_at"`
	Changes    []ChangeLogEntry `json:"changes"`
}

// Sync response
type SyncResponse struct {
	SyncedAt time.Time        `json:"synced_at"`
	Changes  []ChangeLogEntry `json:"changes"`
}

func (api *API) syncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	if err := validateSyncRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := api.db.Begin(r.Context())
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to start transaction", err, map[string]interface{}{
			"user_id":   userID,
			"device_id": req.DeviceID,
		})
		return
	}
	defer tx.Rollback(r.Context())

	var deviceUserID int
	var lastSyncAt *time.Time
	err = tx.QueryRow(r.Context(), `
		SELECT user_id, last_sync_at
		FROM devices
		WHERE id = $1
	`, req.DeviceID).Scan(&deviceUserID, &lastSyncAt)

	if err == pgx.ErrNoRows {
		api.logger.Warn("Sync attempted with unknown device", map[string]interface{}{
			"user_id":   userID,
			"device_id": req.DeviceID,
		})
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to verify device", err, map[string]interface{}{
			"user_id":   userID,
			"device_id": req.DeviceID,
		})
		return
	}

	if deviceUserID != userID {
		api.logger.Warn("Sync attempted with device belonging to different user", map[string]interface{}{
			"requesting_user_id": userID,
			"device_user_id":     deviceUserID,
			"device_id":          req.DeviceID,
		})
		http.Error(w, "device does not belong to user", http.StatusForbidden)
		return
	}

	if err := api.changeLogRepo.RecordChanges(r.Context(), req.DeviceID, userID, req.Changes); err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to record changes", err, map[string]interface{}{
			"user_id":       userID,
			"device_id":     req.DeviceID,
			"changes_count": len(req.Changes),
		})
		return
	}

	remoteChanges, err := api.changeLogRepo.FetchChangesSince(r.Context(), userID, req.DeviceID, req.LastSyncAt)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch remote changes", err, map[string]interface{}{
			"user_id":      userID,
			"device_id":    req.DeviceID,
			"last_sync_at": req.LastSyncAt,
		})
		return
	}

	syncedAt := time.Now()
	_, err = tx.Exec(r.Context(), `
		UPDATE devices
		SET last_sync_at = $1
		WHERE id = $2
	`, syncedAt, req.DeviceID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update sync timestamp", err, map[string]interface{}{
			"user_id":   userID,
			"device_id": req.DeviceID,
		})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to commit transaction", err, map[string]interface{}{
			"user_id":   userID,
			"device_id": req.DeviceID,
		})
		return
	}

	api.logger.Info("Sync completed successfully", map[string]interface{}{
		"user_id":              userID,
		"device_id":            req.DeviceID,
		"changes_uploaded":     len(req.Changes),
		"changes_downloaded":   len(remoteChanges),
		"last_sync_at":         lastSyncAt,
	})

	response := SyncResponse{
		SyncedAt: syncedAt,
		Changes:  remoteChanges,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateSyncRequest(req *SyncRequest) error {
	validEntityTypes := map[string]bool{
		"category":          true,
		"template_group":    true,
		"day_template":      true,
		"weekly_schedule":   true,
		"schedule_override": true,
		"day_record":        true,
		"settings":          true,
	}

	validOperations := map[string]bool{
		"create": true,
		"update": true,
		"delete": true,
	}

	for _, change := range req.Changes {
		if !validEntityTypes[change.EntityType] {
			return &ValidationError{Message: "invalid entity_type: " + change.EntityType}
		}

		if !validOperations[change.Operation] {
			return &ValidationError{Message: "invalid operation: " + change.Operation}
		}

		if change.EntityID <= 0 {
			return &ValidationError{Message: "entity_id must be positive"}
		}

		if change.OccurredAt.IsZero() {
			return &ValidationError{Message: "occurred_at is required"}
		}
	}

	return nil
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
