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
	ID                int          `json:"id"`
	UserID            int          `json:"user_id"`
	DayRangeStartTime ScheduleTime `json:"-"`
	DayRangeEndTime   ScheduleTime `json:"-"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

func (r *UserSettingsRepository) Get(ctx context.Context, userID int) (*UserSettings, error) {
	var settings UserSettings
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_settings (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, user_id, day_range_start_time, day_range_end_time, updated_at
	          FROM user_settings WHERE user_id = $1`
	err = r.db.QueryRow(ctx, query, userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.DayRangeStartTime,
		&settings.DayRangeEndTime,
		&settings.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (r *UserSettingsRepository) Update(ctx context.Context, userID int, dayRangeStartTime ScheduleTime, dayRangeEndTimes ...ScheduleTime) (*UserSettings, error) {
	dayRangeEndTime := ScheduleTime(time.Date(0, time.January, 1, 23, 0, 0, 0, time.UTC))
	if len(dayRangeEndTimes) > 0 {
		dayRangeEndTime = dayRangeEndTimes[0]
	}
	var settings UserSettings
	query := `INSERT INTO user_settings (user_id, day_range_start_time, day_range_end_time, updated_at)
	          VALUES ($3, $1, $2, now())
	          ON CONFLICT (user_id) DO UPDATE
	          SET day_range_start_time = EXCLUDED.day_range_start_time,
	              day_range_end_time = EXCLUDED.day_range_end_time,
	              updated_at = now()
	          RETURNING id, user_id, day_range_start_time, day_range_end_time, updated_at`
	err := r.db.QueryRow(ctx, query, time.Time(dayRangeStartTime), time.Time(dayRangeEndTime), userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.DayRangeStartTime,
		&settings.DayRangeEndTime,
		&settings.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &settings, nil
}
