package main

import (
	"encoding/json"
	"net/http"
)

type DeleteAccountInput struct {
	Password string `json:"password"`
}

type DeleteAccountResponse struct {
	Deleted bool `json:"deleted"`
}

func (api *API) deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := userIDFromRequest(r)

	var input DeleteAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	if input.Password == "" {
		http.Error(w, "password required", http.StatusBadRequest)
		return
	}

	if err := api.userRepo.DeleteAccount(r.Context(), userID, input.Password); err != nil {
		if err == ErrInvalidPassword {
			api.logger.Warn("Account deletion failed - invalid password", map[string]interface{}{
				"user_id": userID,
			})
			http.Error(w, "invalid password confirmation", http.StatusUnauthorized)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to delete account", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	api.logger.Info("User account deleted", map[string]interface{}{
		"user_id": userID,
	})

	writeJSON(w, DeleteAccountResponse{Deleted: true})
}
