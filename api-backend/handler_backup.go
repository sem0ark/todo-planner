package main

import (
	"encoding/json"
	"net/http"
)

func (api *API) backupRouter(responseWriter http.ResponseWriter, request *http.Request) {
	userID := userIDFromRequest(request)
	switch request.Method {
	case http.MethodGet:
		backup, err := api.backupRepo.Export(request.Context(), userID)
		if err != nil {
			HTTPError(responseWriter, request, api.logger, http.StatusInternalServerError, "failed to create backup", err, nil)
			return
		}
		responseWriter.Header().Set("Content-Disposition", `attachment; filename="todo-planner-backup.json"`)
		writeJSON(responseWriter, backup)
	case http.MethodPost:
		var backup BackupData
		decoder := json.NewDecoder(request.Body)
		if err := decoder.Decode(&backup); err != nil || backup.Version != 1 {
			http.Error(responseWriter, "invalid backup JSON", http.StatusBadRequest)
			return
		}
		importedEvents, err := api.backupRepo.Import(request.Context(), userID, backup)
		if err != nil {
			HTTPError(responseWriter, request, api.logger, http.StatusInternalServerError, "failed to import backup", err, nil)
			return
		}
		writeJSON(responseWriter, map[string]interface{}{
			"imported":        true,
			"imported_events": importedEvents,
		})
	default:
		methodNotAllowed(responseWriter)
	}
}
