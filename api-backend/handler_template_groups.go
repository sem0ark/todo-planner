package main

import (
	"encoding/json"
	"net/http"
)

type TemplateGroupsResponse struct {
	Groups []PublicTemplateGroup `json:"groups"`
}

type PublicTemplateGroup struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	CreatedAt APITimestamp `json:"created_at"`
	UpdatedAt APITimestamp `json:"updated_at"`
}

func toPublicTemplateGroup(group TemplateGroup) PublicTemplateGroup {
	return PublicTemplateGroup{ID: group.ID, Name: group.Name,
		CreatedAt: APITimestamp(group.CreatedAt), UpdatedAt: APITimestamp(group.UpdatedAt)}
}

func toPublicTemplateGroups(groups []TemplateGroup) []PublicTemplateGroup {
	publicGroups := make([]PublicTemplateGroup, 0, len(groups))
	for _, group := range groups {
		publicGroups = append(publicGroups, toPublicTemplateGroup(group))
	}
	return publicGroups
}

type TemplateGroupDeleteResponse struct {
	Deleted bool `json:"deleted"`
	ID      int  `json:"id"`
}

func (api *API) getTemplateGroupsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	groups, err := api.templateGroupRepo.FindByUser(r.Context(), userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch template groups", err, map[string]interface{}{"user_id": userID})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TemplateGroupsResponse{Groups: toPublicTemplateGroups(groups)})
}

func (api *API) createTemplateGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input TemplateGroupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group, err := api.templateGroupRepo.Create(r.Context(), input, userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to create template group", err, map[string]interface{}{"user_id": userID, "name": input.Name})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toPublicTemplateGroup(*group))
}

func (api *API) updateTemplateGroupHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input TemplateGroupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
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
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update template group", err, map[string]interface{}{"user_id": userID, "group_id": id})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toPublicTemplateGroup(*group))
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
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to delete template group", err, map[string]interface{}{"user_id": userID, "group_id": id})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TemplateGroupDeleteResponse{Deleted: true, ID: id})
}
