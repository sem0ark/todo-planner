package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type DayTemplatesResponse struct {
	Templates []PublicTemplate `json:"templates"`
}

type PublicTemplateBlock struct {
	CategoryID      int          `json:"category_id"`
	StartTime       ScheduleTime `json:"start_time"`
	DurationMinutes int          `json:"duration_minutes"`
}

type PublicTemplate struct {
	ID              int                   `json:"id"`
	Name            string                `json:"name"`
	TemplateGroupID *int                  `json:"template_group_id"`
	Plan            []PublicTemplateBlock `json:"plan"`
	CreatedAt       APITimestamp          `json:"created_at"`
	UpdatedAt       APITimestamp          `json:"updated_at"`
}

type dayTemplateRequest struct {
	Name            string                 `json:"name"`
	TemplateGroupID *int                   `json:"template_group_id"`
	Plan            *[]templatePlanRequest `json:"plan"`
}

type templatePlanRequest struct {
	CategoryID      int          `json:"category_id"`
	StartTime       ScheduleTime `json:"start_time"`
	DurationMinutes int          `json:"duration_minutes"`
}

func toPublicTemplateBlocks(blocks []SnapshotBlock) []PublicTemplateBlock {
	publicBlocks := make([]PublicTemplateBlock, 0, len(blocks))
	for _, block := range blocks {
		publicBlocks = append(publicBlocks, PublicTemplateBlock{
			CategoryID:      block.CategoryID,
			StartTime:       formatScheduleTime(block.StartTime),
			DurationMinutes: block.DurationMinutes,
		})
	}
	return publicBlocks
}

func toPublicTemplate(template DayTemplate) PublicTemplate {
	plan := make([]PublicTemplateBlock, 0)
	if template.CurrentSnapshot != nil {
		plan = toPublicTemplateBlocks(template.CurrentSnapshot.SnapshotBlocks)
	}
	return PublicTemplate{template.ID, template.Name, template.TemplateGroupID, plan,
		APITimestamp(template.CreatedAt), APITimestamp(template.UpdatedAt)}
}

func toPublicTemplates(templates []DayTemplate) []PublicTemplate {
	publicTemplates := make([]PublicTemplate, 0, len(templates))
	for _, template := range templates {
		publicTemplates = append(publicTemplates, toPublicTemplate(template))
	}
	return publicTemplates
}

func decodeDayTemplateRequest(request *http.Request) (DayTemplateInput, error) {
	var publicInput dayTemplateRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&publicInput); err != nil {
		return DayTemplateInput{}, err
	}
	if publicInput.Plan == nil {
		return DayTemplateInput{}, errors.New("plan is required")
	}
	plan := make([]SnapshotBlockInput, 0, len(*publicInput.Plan))
	for _, block := range *publicInput.Plan {
		plan = append(plan, SnapshotBlockInput{CategoryID: block.CategoryID,
			StartTime: block.StartTime, DurationMinutes: block.DurationMinutes})
	}
	return DayTemplateInput{Name: publicInput.Name, TemplateGroupID: publicInput.TemplateGroupID,
		SnapshotBlocks: plan}, nil
}

type DayTemplateDeleteResponse struct {
	Deleted bool `json:"deleted"`
	ID      int  `json:"id"`
}

func (api *API) getDayTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	templates, err := api.dayTemplateRepo.FindByUser(r.Context(), userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch day templates", err, map[string]interface{}{"user_id": userID})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DayTemplatesResponse{Templates: toPublicTemplates(templates)})
}

func (api *API) createDayTemplateHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	input, err := decodeDayTemplateRequest(r)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	template, err := api.dayTemplateRepo.Create(r.Context(), input, userID)
	if err != nil {
		if errors.Is(err, ErrInvalidTemplateBlock) || errors.Is(err, ErrTemplateCategoryNotFound) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrTemplateGroupNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to create day template", err, map[string]interface{}{"user_id": userID, "name": input.Name})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toPublicTemplate(*template)); err != nil {
		api.logger.Error("failed to encode template response", err, nil)
	}
}

func (api *API) updateDayTemplateHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	input, err := decodeDayTemplateRequest(r)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	template, err := api.dayTemplateRepo.Update(r.Context(), id, input, userID)
	if err == ErrDayTemplateNotFound {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		if errors.Is(err, ErrInvalidTemplateBlock) || errors.Is(err, ErrTemplateCategoryNotFound) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrTemplateGroupNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update day template", err, map[string]interface{}{"user_id": userID, "template_id": id})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toPublicTemplate(*template)); err != nil {
		api.logger.Error("failed to encode template response", err, nil)
	}
}

func (api *API) deleteDayTemplateHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err := api.dayTemplateRepo.Delete(r.Context(), id, userID)
	if err == ErrDayTemplateNotFound {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to delete day template", err, map[string]interface{}{"user_id": userID, "template_id": id})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DayTemplateDeleteResponse{Deleted: true, ID: id})
}
