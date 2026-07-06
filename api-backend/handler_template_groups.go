package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type TemplateGroupsResponse struct {
	TemplateGroups []TemplateGroup `json:"template_groups"`
}

type TemplateGroupDeleteResponse struct {
	Deleted bool `json:"deleted"`
	ID      int  `json:"id"`
}

func (api *API) templateGroupsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/template-groups")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			api.getTemplateGroupsHandler(w, r)
		case http.MethodPost:
			api.createTemplateGroupHandler(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil {
		http.Error(w, "invalid template group id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		api.updateTemplateGroupHandler(w, r, id)
	case http.MethodDelete:
		api.deleteTemplateGroupHandler(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *API) getTemplateGroupsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groups, err := api.templateGroupRepo.FindByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TemplateGroupsResponse{TemplateGroups: groups})
}

func (api *API) createTemplateGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input TemplateGroupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group, err := api.templateGroupRepo.Create(r.Context(), input, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

func (api *API) updateTemplateGroupHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input TemplateGroupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group, err := api.templateGroupRepo.Update(r.Context(), id, input, userID)
	if err == ErrTemplateGroupNotFound {
		http.Error(w, "template group not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

func (api *API) deleteTemplateGroupHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err := api.templateGroupRepo.Delete(r.Context(), id, userID)
	if err == ErrTemplateGroupNotFound {
		http.Error(w, "template group not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TemplateGroupDeleteResponse{Deleted: true, ID: id})
}
