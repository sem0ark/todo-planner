package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5"
)

var (
	ErrCategoryNameRequired     = errors.New("name is required")
	ErrInvalidCategoryColor     = errors.New("invalid color format")
	ErrInvalidPomodoroDurations = errors.New("pomodoro durations must be positive")
)

type PomodoroConfig struct {
	WorkDuration int `json:"work_duration"`
	RestDuration int `json:"rest_duration"`
}

type CategoryInput struct {
	Name           string          `json:"name"`
	Color          string          `json:"color"`
	PomodoroConfig *PomodoroConfig `json:"pomodoro_config"`
}

type CategoriesResponse struct {
	Categories []BlockCategory `json:"categories"`
}

type CategoryDeleteResponse struct {
	ID      int  `json:"id"`
	Deleted bool `json:"deleted"`
}

func (api *API) getCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	categories, err := api.categoryRepo.FindByUser(r.Context(), userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to fetch categories", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	response := CategoriesResponse{Categories: categories}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *API) createCategoryHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	if err := validateCategoryInput(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	category, err := api.categoryRepo.Create(r.Context(), input, userID)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to create category", err, map[string]interface{}{
			"user_id": userID,
			"name":    input.Name,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(category)
}

func (api *API) updateCategoryHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, map[string]interface{}{
			"user_id": userID,
		})
		return
	}

	if err := validateCategoryInput(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	category, err := api.categoryRepo.Update(r.Context(), id, input, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to update category", err, map[string]interface{}{
			"user_id":     userID,
			"category_id": id,
			"name":        input.Name,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(category)
}

func (api *API) deleteCategoryHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err := api.categoryRepo.Delete(r.Context(), id, userID)
	if err != nil {
		if err == ErrCategoryNotFound {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to delete category", err, map[string]interface{}{
			"user_id":     userID,
			"category_id": id,
		})
		return
	}

	response := CategoryDeleteResponse{ID: id, Deleted: true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateCategoryInput(input CategoryInput) error {
	if input.Name == "" {
		return ErrCategoryNameRequired
	}
	if !isValidHexColor(input.Color) {
		return ErrInvalidCategoryColor
	}
	return validatePomodoroConfig(input.PomodoroConfig)
}

func validatePomodoroConfig(config *PomodoroConfig) error {
	if config == nil {
		return nil
	}
	if config.WorkDuration <= 0 || config.RestDuration <= 0 {
		return ErrInvalidPomodoroDurations
	}
	return nil
}

func isValidHexColor(color string) bool {
	matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, color)
	return matched
}
