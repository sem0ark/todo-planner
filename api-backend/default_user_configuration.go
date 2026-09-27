package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

//go:embed default_user_configuration.json
var defaultUserConfigurationJSON []byte

type DefaultUserConfiguration struct {
	Version        int                            `json:"version"`
	Settings       DefaultSettingsConfiguration   `json:"settings"`
	Categories     []DefaultCategoryConfiguration `json:"categories"`
	Templates      []DefaultTemplateConfiguration `json:"templates"`
	WeeklySchedule []DefaultWeeklyScheduleEntry   `json:"weekly_schedule"`
}

type DefaultSettingsConfiguration struct {
	DayRangeStartTime string `json:"day_range_start_time"`
	DayRangeEndTime   string `json:"day_range_end_time"`
}

type DefaultCategoryConfiguration struct {
	Key            string          `json:"key"`
	Name           string          `json:"name"`
	Color          string          `json:"color"`
	PomodoroConfig *PomodoroConfig `json:"pomodoro_config"`
}

type DefaultTemplateConfiguration struct {
	Key  string                     `json:"key"`
	Name string                     `json:"name"`
	Plan []DefaultPlanConfiguration `json:"plan,omitempty"`
}

type DefaultPlanConfiguration struct {
	CategoryKey     string `json:"category_key"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
}

type DefaultWeeklyScheduleEntry struct {
	DayOfWeek   int    `json:"day_of_week"`
	TemplateKey string `json:"template_key"`
}

func loadDefaultUserConfiguration() (DefaultUserConfiguration, error) {
	var configuration DefaultUserConfiguration
	if err := json.Unmarshal(defaultUserConfigurationJSON, &configuration); err != nil {
		return configuration, fmt.Errorf("decode default user configuration: %w", err)
	}
	if err := validateDefaultUserConfiguration(configuration); err != nil {
		return configuration, err
	}
	return configuration, nil
}

func validateDefaultUserConfiguration(configuration DefaultUserConfiguration) error {
	if configuration.Version != 1 {
		return fmt.Errorf("unsupported default user configuration version: %d", configuration.Version)
	}
	categoryKeys := make(map[string]struct{}, len(configuration.Categories))
	for _, category := range configuration.Categories {
		if category.Key == "" || category.Name == "" {
			return fmt.Errorf("default category key and name are required")
		}
		if _, exists := categoryKeys[category.Key]; exists {
			return fmt.Errorf("duplicate default category key: %s", category.Key)
		}
		categoryKeys[category.Key] = struct{}{}
	}
	templateKeys := make(map[string]struct{}, len(configuration.Templates))
	for _, template := range configuration.Templates {
		if template.Key == "" || template.Name == "" {
			return fmt.Errorf("default template key and name are required")
		}
		if _, exists := templateKeys[template.Key]; exists {
			return fmt.Errorf("duplicate default template key: %s", template.Key)
		}
		templateKeys[template.Key] = struct{}{}
		for _, block := range template.Plan {
			if _, exists := categoryKeys[block.CategoryKey]; !exists {
				return fmt.Errorf("unknown default category key: %s", block.CategoryKey)
			}
		}
	}
	if len(configuration.WeeklySchedule) != 7 {
		return fmt.Errorf("default weekly schedule must contain seven entries")
	}
	seenDays := make(map[int]struct{}, len(configuration.WeeklySchedule))
	for _, entry := range configuration.WeeklySchedule {
		if entry.DayOfWeek < 0 || entry.DayOfWeek > 6 {
			return fmt.Errorf("invalid default day of week: %d", entry.DayOfWeek)
		}
		if _, exists := seenDays[entry.DayOfWeek]; exists {
			return fmt.Errorf("duplicate default day of week: %d", entry.DayOfWeek)
		}
		seenDays[entry.DayOfWeek] = struct{}{}
		if _, exists := templateKeys[entry.TemplateKey]; !exists {
			return fmt.Errorf("unknown default template key: %s", entry.TemplateKey)
		}
	}
	return nil
}

func createDefaultUserConfiguration(ctx context.Context, transaction pgx.Tx, userID int) error {
	configuration, err := loadDefaultUserConfiguration()
	if err != nil {
		return err
	}
	return NewBackupRepository(nil).ImportConfiguration(ctx, transaction, userID, configuration)
}

func applyUserConfiguration(ctx context.Context, transaction pgx.Tx, userID int, configuration DefaultUserConfiguration) error {
	var err error
	categoryIDs := make(map[string]int, len(configuration.Categories))
	for _, category := range configuration.Categories {
		var categoryID int
		err = transaction.QueryRow(ctx, `
			INSERT INTO block_categories (user_id, name, color, pomodoro_config)
			VALUES ($1, $2, $3, $4) RETURNING id
		`, userID, category.Name, category.Color, category.PomodoroConfig).Scan(&categoryID)
		if err != nil {
			return err
		}
		categoryIDs[category.Key] = categoryID
	}

	templateIDs := make(map[string]int, len(configuration.Templates))
	for _, template := range configuration.Templates {
		var templateID, snapshotID int
		err = transaction.QueryRow(ctx, `
			INSERT INTO day_templates (user_id, name) VALUES ($1, $2) RETURNING id
		`, userID, template.Name).Scan(&templateID)
		if err != nil {
			return err
		}
		err = transaction.QueryRow(ctx, `
			INSERT INTO template_snapshots (day_template_id, user_id)
			VALUES ($1, $2) RETURNING id
		`, templateID, userID).Scan(&snapshotID)
		if err != nil {
			return err
		}
		for _, block := range template.Plan {
			_, err = transaction.Exec(ctx, `
				INSERT INTO snapshot_blocks (snapshot_id, category_id, start_time, duration_minutes)
				VALUES ($1, $2, $3, $4)
			`, snapshotID, categoryIDs[block.CategoryKey], block.StartTime, block.DurationMinutes)
			if err != nil {
				return err
			}
		}
		templateIDs[template.Key] = templateID
	}

	for _, entry := range configuration.WeeklySchedule {
		var templateID *int
		if entry.TemplateKey != "" {
			resolvedTemplateID := templateIDs[entry.TemplateKey]
			templateID = &resolvedTemplateID
		}
		_, err = transaction.Exec(ctx, `
			INSERT INTO weekly_schedule (user_id, day_of_week, day_template_id)
			VALUES ($1, $2, $3)
		`, userID, entry.DayOfWeek, templateID)
		if err != nil {
			return err
		}
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO user_settings (user_id, day_range_start_time, day_range_end_time, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			day_range_start_time = EXCLUDED.day_range_start_time,
			day_range_end_time = EXCLUDED.day_range_end_time,
			updated_at = NOW()
	`, userID, configuration.Settings.DayRangeStartTime, configuration.Settings.DayRangeEndTime)
	return err
}
