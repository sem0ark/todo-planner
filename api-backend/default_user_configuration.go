package main

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func createDefaultUserConfiguration(ctx context.Context, tx pgx.Tx, userID int) error {
	categoryIDs := make(map[string]int, 4)
	for _, category := range []struct {
		name           string
		color          string
		pomodoroConfig *string
	}{
		{name: "Working", color: "#2563eb", pomodoroConfig: stringPointer(`{"work_duration":2700,"rest_duration":300}`)},
		{name: "Exercise", color: "#dc2626"},
		{name: "Rest", color: "#0891b2"},
		{name: "Learning", color: "#27b208"},
	} {
		var categoryID int
		err := tx.QueryRow(ctx, `
			INSERT INTO block_categories (user_id, name, color, pomodoro_config)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, userID, category.name, category.color, category.pomodoroConfig).Scan(&categoryID)
		if err != nil {
			return err
		}
		categoryIDs[category.name] = categoryID
	}

	var templateID int
	err := tx.QueryRow(ctx, `
		INSERT INTO day_templates (user_id, name)
		VALUES ($1, 'Working Day')
		RETURNING id
	`, userID).Scan(&templateID)
	if err != nil {
		return err
	}

	blocks := []struct {
		categoryName string
		startTime    string
		duration     int
	}{
		{categoryName: "Rest", startTime: "00:00:00", duration: 480},
		{categoryName: "Learning", startTime: "08:00:00", duration: 60},
		{categoryName: "Working", startTime: "09:00:00", duration: 180},
		{categoryName: "Rest", startTime: "12:00:00", duration: 60},
		{categoryName: "Working", startTime: "13:00:00", duration: 300},
		{categoryName: "Exercise", startTime: "18:00:00", duration: 60},
		{categoryName: "Rest", startTime: "19:00:00", duration: 360},
	}

	for _, block := range blocks {
		_, err = tx.Exec(ctx, `
			INSERT INTO planned_blocks (day_template_id, category_id, start_time, duration_minutes)
			VALUES ($1, $2, $3, $4)
		`, templateID, categoryIDs[block.categoryName], block.startTime, block.duration)
		if err != nil {
			return err
		}
	}

	var snapshotID int
	err = tx.QueryRow(ctx, `
		INSERT INTO template_snapshots (day_template_id, user_id)
		VALUES ($1, $2)
		RETURNING id
	`, templateID, userID).Scan(&snapshotID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO snapshot_blocks (snapshot_id, category_id, start_time, duration_minutes)
		SELECT $1, category_id, start_time, duration_minutes
		FROM planned_blocks
		WHERE day_template_id = $2
	`, snapshotID, templateID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO weekly_schedule (user_id, day_of_week, day_template_id)
		SELECT $1, day_of_week, $2
		FROM generate_series(0, 6) AS days(day_of_week)
	`, userID, templateID)
	return err
}

func stringPointer(value string) *string {
	return &value
}