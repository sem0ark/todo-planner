package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Category creation/update request data
type CategoryInput struct {
	Name  string `json:"name"`
	Color string `json:"color"` // hex color
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if !isValidHexColor(input.Color) {
		http.Error(w, "invalid color format", http.StatusBadRequest)
		return
	}

	category, err := api.categoryRepo.Create(r.Context(), input, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if !isValidHexColor(input.Color) {
		http.Error(w, "invalid color format", http.StatusBadRequest)
		return
	}

	category, err := api.categoryRepo.Update(r.Context(), id, input, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := CategoryDeleteResponse{
		ID:      id,
		Deleted: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *API) categoriesHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/categories")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			api.getCategoriesHandler(w, r)
		case http.MethodPost:
			api.createCategoryHandler(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil {
		http.Error(w, "invalid category ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		api.updateCategoryHandler(w, r, id)
	case http.MethodDelete:
		api.deleteCategoryHandler(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func isValidHexColor(color string) bool {
	matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, color)
	return matched
}
