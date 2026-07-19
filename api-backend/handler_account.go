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

	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input DeleteAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Password == "" {
		http.Error(w, "password required", http.StatusBadRequest)
		return
	}

	if err := api.userRepo.DeleteAccount(r.Context(), userID, input.Password); err != nil {
		if err == ErrInvalidPassword {
			http.Error(w, "invalid password confirmation", http.StatusUnauthorized)
			return
		}
		http.Error(w, "failed to delete account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DeleteAccountResponse{Deleted: true})
}
