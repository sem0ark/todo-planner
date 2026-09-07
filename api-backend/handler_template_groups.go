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
	userID := userIDFromRequest(r)

	groups, err := api.templateGroupRepo.FindByUser(r.Context(), userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch template groups", err, map[string]interface{}{"user_id": userID})
		return
	}

	writeJSON(w, TemplateGroupsResponse{Groups: toPublicTemplateGroups(groups)})
}

func (api *API) createTemplateGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromRequest(r)

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

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, toPublicTemplateGroup(*group))
}

func (api *API) updateTemplateGroupHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID := userIDFromRequest(r)

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
	if err != nil {
		if writeAppError(w, err) {
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update template group", err, map[string]interface{}{"user_id": userID, "group_id": id})
		return
	}

	writeJSON(w, toPublicTemplateGroup(*group))
}

func (api *API) deleteTemplateGroupHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID := userIDFromRequest(r)

	err := api.templateGroupRepo.Delete(r.Context(), id, userID)
	if err != nil {
		if writeAppError(w, err) {
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to delete template group", err, map[string]interface{}{"user_id": userID, "group_id": id})
		return
	}

	writeJSON(w, TemplateGroupDeleteResponse{Deleted: true, ID: id})
}
