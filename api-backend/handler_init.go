package main

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type initRequest struct {
	DeviceID     int    `json:"device_id"`
	CalendarDate string `json:"calendar_date"`
}

type initResponse struct {
	Settings   *UserSettings   `json:"settings"`
	Categories []BlockCategory `json:"categories"`
	DayRecord  publicDayRecord `json:"day_record"`
}

// initHandler returns the small bootstrap payload required by a native client.
func (api *API) initHandler(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, authenticated := getUserID(request.Context())
	if !authenticated {
		http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
		return
	}
	var input initRequest
	if json.NewDecoder(request.Body).Decode(&input) != nil || input.DeviceID <= 0 || !isValidCalendarDate(input.CalendarDate) {
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	transaction, err := api.db.Begin(request.Context())
	if err != nil {
		http.Error(responseWriter, "failed to initialize client", http.StatusInternalServerError)
		return
	}
	defer transaction.Rollback(request.Context())
	var deviceUserID int
	err = transaction.QueryRow(request.Context(), `SELECT user_id FROM devices WHERE id = $1`, input.DeviceID).Scan(&deviceUserID)
	if err != nil || deviceUserID != userID {
		http.Error(responseWriter, "device not found", http.StatusNotFound)
		return
	}
	settings, err := loadSettingsForInit(request, transaction, userID)
	if err != nil {
		http.Error(responseWriter, "failed to load settings", http.StatusInternalServerError)
		return
	}
	categories, err := loadCategoriesForInit(request, transaction, userID)
	if err != nil {
		http.Error(responseWriter, "failed to load categories", http.StatusInternalServerError)
		return
	}
	dayRecordID, err := findOrCreateDayRecord(request.Context(), transaction, userID, input.CalendarDate)
	if err != nil {
		http.Error(responseWriter, "failed to load day", http.StatusInternalServerError)
		return
	}
	if err := transaction.Commit(request.Context()); err != nil {
		http.Error(responseWriter, "failed to initialize client", http.StatusInternalServerError)
		return
	}
	record, err := api.dayRecordRepo.FindByID(request.Context(), dayRecordID, userID)
	if err != nil {
		http.Error(responseWriter, "failed to load day", http.StatusInternalServerError)
		return
	}
	writeJSON(responseWriter, initResponse{
		Settings:   settings,
		Categories: categories,
		DayRecord:  toPublicDayRecord(record),
	})
}

func loadSettingsForInit(request *http.Request, transaction pgx.Tx, userID int) (*UserSettings, error) {
	var settings UserSettings
	err := transaction.QueryRow(
		request.Context(),
		`SELECT id, user_id, day_boundary_time::time(0)::text, updated_at FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&settings.ID, &settings.UserID, &settings.DayBoundaryTime, &settings.UpdatedAt)
	if err == pgx.ErrNoRows {
		err = transaction.QueryRow(
			request.Context(),
			`INSERT INTO user_settings(user_id) VALUES($1) RETURNING id, user_id, day_boundary_time::time(0)::text, updated_at`, userID,
		).Scan(&settings.ID, &settings.UserID, &settings.DayBoundaryTime, &settings.UpdatedAt)
	}
	return &settings, err
}

func loadCategoriesForInit(request *http.Request, transaction pgx.Tx, userID int) ([]BlockCategory, error) {
	rows, err := transaction.Query(request.Context(), `SELECT id, user_id, name, color, pomodoro_config, created_at, updated_at FROM block_categories WHERE user_id = $1 AND is_deleted = FALSE ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := make([]BlockCategory, 0)
	for rows.Next() {
		var category BlockCategory
		if err := rows.Scan(&category.ID, &category.UserID, &category.Name, &category.Color, &category.PomodoroConfig, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}
