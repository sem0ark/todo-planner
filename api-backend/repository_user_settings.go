package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserSettingsRepository struct {
	db *pgxpool.Pool
}

func NewUserSettingsRepository(db *pgxpool.Pool) *UserSettingsRepository {
	return &UserSettingsRepository{db: db}
}

type UserSettings struct {
	ID              int          `json:"id"`
	UserID          int          `json:"user_id"`
	DayBoundaryTime ScheduleTime `json:"-"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

func (r *UserSettingsRepository) GetOrCreate(ctx context.Context, userID int) (*UserSettings, error) {
	var settings UserSettings
	query := `SELECT id, user_id, day_boundary_time, updated_at
	          FROM user_settings WHERE user_id = $1`
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.DayBoundaryTime,
		&settings.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return r.create(ctx, userID)
		}
		return nil, err
	}

	return &settings, nil
}

func (r *UserSettingsRepository) create(ctx context.Context, userID int) (*UserSettings, error) {
	var settings UserSettings
	defaultBoundaryTime := time.Date(0, time.January, 1, 4, 0, 0, 0, time.UTC)
	query := `INSERT INTO user_settings (user_id, day_boundary_time, updated_at)
	          VALUES ($1, $2, now())
	          RETURNING id, user_id, day_boundary_time, updated_at`
	err := r.db.QueryRow(ctx, query, userID, defaultBoundaryTime).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.DayBoundaryTime,
		&settings.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (r *UserSettingsRepository) Update(ctx context.Context, userID int, dayBoundaryTime ScheduleTime) (*UserSettings, error) {
	var settings UserSettings
	query := `UPDATE user_settings
	          SET day_boundary_time = $1, updated_at = now()
	          WHERE user_id = $2
	          RETURNING id, user_id, day_boundary_time, updated_at`
	err := r.db.QueryRow(ctx, query, time.Time(dayBoundaryTime), userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.DayBoundaryTime,
		&settings.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &settings, nil
}
