package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type DayTemplatesResponse struct {
	Templates []DayTemplate `json:"templates"`
}

type DayTemplateDeleteResponse struct {
	Deleted bool `json:"deleted"`
	ID      int  `json:"id"`
}

func (api *API) dayTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/templates")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			api.getDayTemplatesHandler(w, r)
		case http.MethodPost:
			api.createDayTemplateHandler(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil {
		http.Error(w, "invalid template id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		api.updateDayTemplateHandler(w, r, id)
	case http.MethodDelete:
		api.deleteDayTemplateHandler(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *API) getDayTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	templates, err := api.dayTemplateRepo.FindByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DayTemplatesResponse{Templates: templates})
}

func (api *API) createDayTemplateHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input DayTemplateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	template, err := api.dayTemplateRepo.Create(r.Context(), input, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func (api *API) updateDayTemplateHandler(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := getUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input DayTemplateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	template, err := api.dayTemplateRepo.Update(r.Context(), id, input, userID)
	if err == ErrDayTemplateNotFound {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DayTemplateDeleteResponse{Deleted: true, ID: id})
}
