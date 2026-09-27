package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BackupEvent struct {
	ID                  int          `json:"id"`
	CalendarDate        CalendarDate `json:"calendar_date"`
	DeviceID            *int         `json:"device_id,omitempty"`
	ClientEventID       *string      `json:"client_event_id,omitempty"`
	EventType           string       `json:"event_type"`
	CategoryID          *int         `json:"category_id"`
	OccurredAt          time.Time    `json:"occurred_at"`
	OccurredAtLocal     string       `json:"occurred_at_local"`
	TargetClientEventID *string      `json:"target_client_event_id,omitempty"`
	CorrectedAt         *time.Time   `json:"corrected_at,omitempty"`
	CorrectedAtLocal    *string      `json:"corrected_at_local,omitempty"`
}

type BackupData struct {
	Version        int                `json:"version"`
	ExportedAt     *time.Time         `json:"exported_at,omitempty"`
	Settings       *BackupSettings    `json:"settings"`
	Categories     []BlockCategory    `json:"categories"`
	Groups         []TemplateGroup    `json:"template_groups"`
	Templates      []BackupTemplate   `json:"templates"`
	WeeklySchedule []WeeklySchedule   `json:"weekly_schedule"`
	Overrides      []ScheduleOverride `json:"overrides"`
	Days           []BackupDay        `json:"days"`
	Events         []BackupEvent      `json:"events"`
}

type BackupDay struct {
	CalendarDate          CalendarDate        `json:"calendar_date"`
	DayTemplateID         *int                `json:"day_template_id,omitempty"`
	SnapshotID            *int                `json:"snapshot_id,omitempty"`
	TimezoneOffsetMinutes *int                `json:"timezone_offset_minutes,omitempty"`
	TimezoneOffsetLocked  bool                `json:"timezone_offset_locked"`
	Plan                  []BackupPlanBlock   `json:"plan,omitempty"`
	Actual                []BackupActualBlock `json:"actual,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

type BackupPlanBlock struct {
	CategoryID      int    `json:"category_id"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
}

type BackupTemplate struct {
	ID              int               `json:"id"`
	Name            string            `json:"name"`
	TemplateGroupID *int              `json:"template_group_id,omitempty"`
	Plan            []BackupPlanBlock `json:"plan,omitempty"`
}

type BackupActualBlock struct {
	CategoryID      *int   `json:"category_id"`
	BlockType       string `json:"block_type"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
}

type BackupSettings struct {
	DayRangeStartTime string    `json:"day_range_start_time"`
	DayRangeEndTime   string    `json:"day_range_end_time"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewBackupRepository(db *pgxpool.Pool) *BackupRepository {
	return &BackupRepository{db: db}
}

type BackupRepository struct{ db *pgxpool.Pool }

// ImportConfiguration applies a symbolic configuration inside an existing transaction.
// Default-user creation and backup restoration use the same persistence path.
func (repository *BackupRepository) ImportConfiguration(ctx context.Context, transaction pgx.Tx, userID int, configuration DefaultUserConfiguration) error {
	return applyUserConfiguration(ctx, transaction, userID, configuration)
}

func (repository *BackupRepository) Export(ctx context.Context, userID int) (*BackupData, error) {
	settings, err := NewUserSettingsRepository(repository.db).Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	categories, err := NewCategoryRepository(repository.db).FindByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	groups, err := NewTemplateGroupRepository(repository.db).FindByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	templates, err := NewDayTemplateRepository(repository.db).FindByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	weeklySchedule, err := NewScheduleRepository(repository.db).GetWeeklySchedule(ctx, userID)
	if err != nil {
		return nil, err
	}
	overrides, err := repository.findAllOverrides(ctx, userID)
	if err != nil {
		return nil, err
	}
	records, err := NewDayRecordRepository(repository.db).FindByDateRange(ctx, userID, CalendarDate(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)), CalendarDate(time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		return nil, err
	}
	events, err := repository.findEvents(ctx, userID)
	if err != nil {
		return nil, err
	}
	exportedAt := time.Now().UTC()
	backupSettings := &BackupSettings{
		DayRangeStartTime: backupScheduleTime(settings.DayRangeStartTime),
		DayRangeEndTime:   backupScheduleTime(settings.DayRangeEndTime),
		UpdatedAt:         settings.UpdatedAt,
	}
	backupDays := make([]BackupDay, 0, len(records))
	for _, record := range records {
		backupDay := BackupDay{
			CalendarDate: record.CalendarDate, DayTemplateID: record.DayTemplateID,
			SnapshotID: record.SnapshotID, TimezoneOffsetMinutes: record.TimezoneOffsetMinutes,
			TimezoneOffsetLocked: record.TimezoneOffsetLocked, CreatedAt: record.CreatedAt,
			UpdatedAt: record.UpdatedAt, Plan: make([]BackupPlanBlock, 0, len(record.SnapshotBlocks)),
			Actual: make([]BackupActualBlock, 0, len(record.ActualBlocks)),
		}
		for _, block := range record.SnapshotBlocks {
			backupDay.Plan = append(backupDay.Plan, BackupPlanBlock{CategoryID: block.CategoryID, StartTime: time.Time(block.StartTime).Format(ScheduleTimeFormat), DurationMinutes: block.DurationMinutes})
		}
		for _, block := range record.ActualBlocks {
			backupDay.Actual = append(backupDay.Actual, BackupActualBlock{CategoryID: block.CategoryID, BlockType: block.BlockType, StartTime: time.Time(block.StartTime).Format(ScheduleTimeFormat), DurationMinutes: block.DurationMinutes})
		}
		backupDays = append(backupDays, backupDay)
	}
	backupTemplates := make([]BackupTemplate, 0, len(templates))
	for _, template := range templates {
		backupTemplate := BackupTemplate{ID: template.ID, Name: template.Name, TemplateGroupID: template.TemplateGroupID, Plan: make([]BackupPlanBlock, 0)}
		if template.CurrentSnapshot != nil {
			backupTemplate.Plan = make([]BackupPlanBlock, 0, len(template.CurrentSnapshot.SnapshotBlocks))
			for _, block := range template.CurrentSnapshot.SnapshotBlocks {
				backupTemplate.Plan = append(backupTemplate.Plan, BackupPlanBlock{CategoryID: block.CategoryID, StartTime: time.Time(block.StartTime).Format(ScheduleTimeFormat), DurationMinutes: block.DurationMinutes})
			}
		}
		backupTemplates = append(backupTemplates, backupTemplate)
	}
	return &BackupData{Version: 1, ExportedAt: &exportedAt, Settings: backupSettings, Categories: categories, Groups: groups, Templates: backupTemplates, WeeklySchedule: weeklySchedule, Overrides: overrides, Days: backupDays, Events: events}, nil
}

func (repository *BackupRepository) findAllOverrides(ctx context.Context, userID int) ([]ScheduleOverride, error) {
	rows, err := repository.db.Query(ctx, `SELECT id, user_id, calendar_date, day_template_id, created_at FROM schedule_overrides WHERE user_id = $1 ORDER BY calendar_date`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	overrides := make([]ScheduleOverride, 0)
	for rows.Next() {
		var override ScheduleOverride
		if err := rows.Scan(&override.ID, &override.UserID, &override.CalendarDate, &override.DayTemplateID, &override.CreatedAt); err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	return overrides, rows.Err()
}

func backupScheduleTime(value ScheduleTime) string {
	return time.Time(value).Format("15:04:05")
}

func (repository *BackupRepository) findEvents(ctx context.Context, userID int) ([]BackupEvent, error) {
	rows, err := repository.db.Query(ctx, `
		SELECT events.id, records.calendar_date, events.device_id, events.client_event_id,
		       events.event_type, events.category_id, events.occurred_at,
		       COALESCE(events.occurred_at_local, ''), events.target_client_event_id,
		       events.corrected_at, events.corrected_at_local
		FROM day_events events
		JOIN day_records records ON records.id = events.day_record_id
		WHERE records.user_id = $1
		ORDER BY records.calendar_date, events.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]BackupEvent, 0)
	for rows.Next() {
		var event BackupEvent
		if err := rows.Scan(&event.ID, &event.CalendarDate, &event.DeviceID, &event.ClientEventID, &event.EventType, &event.CategoryID, &event.OccurredAt, &event.OccurredAtLocal, &event.TargetClientEventID, &event.CorrectedAt, &event.CorrectedAtLocal); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (repository *BackupRepository) Import(ctx context.Context, userID int, backup BackupData) (int, error) {
	transaction, err := repository.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer transaction.Rollback(ctx)

	if backup.Settings != nil {
		configuration := DefaultUserConfiguration{
			Version: 1,
			Settings: DefaultSettingsConfiguration{
				DayRangeStartTime: backup.Settings.DayRangeStartTime,
				DayRangeEndTime:   backup.Settings.DayRangeEndTime,
			},
		}
		if err := repository.ImportConfiguration(ctx, transaction, userID, configuration); err != nil {
			return 0, err
		}
	}
	categoryIDs := make(map[int]int, len(backup.Categories))
	for _, category := range backup.Categories {
		var categoryID int
		err = transaction.QueryRow(ctx, `SELECT id FROM block_categories WHERE user_id = $1 AND name = $2 AND is_deleted = FALSE LIMIT 1`, userID, category.Name).Scan(&categoryID)
		if err == pgx.ErrNoRows {
			err = transaction.QueryRow(ctx, `INSERT INTO block_categories (user_id, name, color, pomodoro_config) VALUES ($1, $2, $3, $4) RETURNING id`, userID, category.Name, category.Color, category.PomodoroConfig).Scan(&categoryID)
		}
		if err != nil {
			return 0, err
		}
		categoryIDs[category.ID] = categoryID
	}
	groupIDs := make(map[int]int, len(backup.Groups))
	for _, group := range backup.Groups {
		var groupID int
		err = transaction.QueryRow(ctx, `SELECT id FROM template_groups WHERE user_id = $1 AND name = $2 AND is_deleted = FALSE LIMIT 1`, userID, group.Name).Scan(&groupID)
		if err == pgx.ErrNoRows {
			err = transaction.QueryRow(ctx, `INSERT INTO template_groups (user_id, name) VALUES ($1, $2) RETURNING id`, userID, group.Name).Scan(&groupID)
		}
		if err != nil {
			return 0, err
		}
		groupIDs[group.ID] = groupID
	}
	templateIDs := make(map[int]int, len(backup.Templates))
	for _, template := range backup.Templates {
		var templateID int
		var groupID *int
		if template.TemplateGroupID != nil {
			mappedGroupID, exists := groupIDs[*template.TemplateGroupID]
			if !exists {
				return 0, fmt.Errorf("backup template group %d is missing", *template.TemplateGroupID)
			}
			groupID = &mappedGroupID
		}
		err = transaction.QueryRow(ctx, `SELECT id FROM day_templates WHERE user_id = $1 AND name = $2 AND is_deleted = FALSE LIMIT 1`, userID, template.Name).Scan(&templateID)
		if err == pgx.ErrNoRows {
			err = transaction.QueryRow(ctx, `INSERT INTO day_templates (user_id, template_group_id, name) VALUES ($1, $2, $3) RETURNING id`, userID, groupID, template.Name).Scan(&templateID)
		}
		if err != nil {
			return 0, err
		}
		templateIDs[template.ID] = templateID
		if _, err := repository.insertTemplateSnapshot(ctx, transaction, userID, templateID, template.Plan, categoryIDs); err != nil {
			return 0, err
		}
	}
	for _, scheduleEntry := range backup.WeeklySchedule {
		var templateID *int
		if scheduleEntry.DayTemplateID != nil {
			mappedTemplateID, exists := templateIDs[*scheduleEntry.DayTemplateID]
			if !exists {
				return 0, fmt.Errorf("backup schedule template %d is missing", *scheduleEntry.DayTemplateID)
			}
			templateID = &mappedTemplateID
		}
		_, err = transaction.Exec(ctx, `INSERT INTO weekly_schedule (user_id, day_of_week, day_template_id) VALUES ($1, $2, $3) ON CONFLICT (user_id, day_of_week) DO UPDATE SET day_template_id = EXCLUDED.day_template_id, updated_at = NOW()`, userID, scheduleEntry.DayOfWeek, templateID)
		if err != nil {
			return 0, err
		}
	}
	for _, override := range backup.Overrides {
		if override.DayTemplateID == nil {
			continue
		}
		mappedTemplateID, exists := templateIDs[*override.DayTemplateID]
		if !exists {
			return 0, fmt.Errorf("backup override template %d is missing", *override.DayTemplateID)
		}
		_, err = transaction.Exec(ctx, `INSERT INTO schedule_overrides (user_id, calendar_date, day_template_id) VALUES ($1, $2, $3) ON CONFLICT (user_id, calendar_date) DO UPDATE SET day_template_id = EXCLUDED.day_template_id`, userID, override.CalendarDate, mappedTemplateID)
		if err != nil {
			return 0, err
		}
	}

	importedEvents := 0
	dayRecordIDs := make(map[int]struct{})
	for _, day := range backup.Days {
		var dayRecordID int
		var templateID *int
		if day.DayTemplateID != nil {
			mappedTemplateID, exists := templateIDs[*day.DayTemplateID]
			if !exists {
				return 0, fmt.Errorf("backup day template %d is missing", *day.DayTemplateID)
			}
			templateID = &mappedTemplateID
		}
		var snapshotID *int
		if templateID != nil {
			createdSnapshotID, snapshotError := repository.insertTemplateSnapshot(ctx, transaction, userID, *templateID, day.Plan, categoryIDs)
			if snapshotError != nil {
				return 0, snapshotError
			}
			snapshotID = &createdSnapshotID
		}
		err = transaction.QueryRow(ctx, `SELECT id FROM day_records WHERE user_id = $1 AND calendar_date = $2`, userID, day.CalendarDate).Scan(&dayRecordID)
		if err == pgx.ErrNoRows {
			err = transaction.QueryRow(ctx, `INSERT INTO day_records (user_id, day_template_id, snapshot_id, calendar_date) VALUES ($1, $2, $3, $4) RETURNING id`, userID, templateID, snapshotID, day.CalendarDate).Scan(&dayRecordID)
		}
		if err != nil {
			return 0, err
		}
		dayRecordIDs[dayRecordID] = struct{}{}
		_, err = transaction.Exec(ctx, `UPDATE day_records SET day_template_id = COALESCE($1, day_template_id), snapshot_id = COALESCE($2, snapshot_id), timezone_offset_minutes = $3, timezone_offset_locked = $4, updated_at = NOW() WHERE id = $5 AND user_id = $6`, templateID, snapshotID, day.TimezoneOffsetMinutes, day.TimezoneOffsetLocked, dayRecordID, userID)
		if err != nil {
			return 0, err
		}
		_, err = transaction.Exec(ctx, `DELETE FROM actual_blocks WHERE day_record_id = $1`, dayRecordID)
		if err != nil {
			return 0, err
		}
		for _, block := range day.Actual {
			var categoryID *int
			if block.CategoryID != nil {
				mappedCategoryID, exists := categoryIDs[*block.CategoryID]
				if !exists {
					mappedCategoryID = *block.CategoryID
				}
				categoryID = &mappedCategoryID
			}
			_, err = transaction.Exec(ctx, `INSERT INTO actual_blocks (day_record_id, category_id, block_type, start_time, duration_minutes, updated_at) VALUES ($1, $2, $3, $4, $5, NOW())`, dayRecordID, categoryID, block.BlockType, block.StartTime, block.DurationMinutes)
			if err != nil {
				return 0, err
			}
		}
	}
	for _, event := range backup.Events {
		var dayRecordID int
		err = transaction.QueryRow(ctx, `
			SELECT id FROM day_records WHERE user_id = $1 AND calendar_date = $2
		`, userID, event.CalendarDate).Scan(&dayRecordID)
		if err == pgx.ErrNoRows {
			err = transaction.QueryRow(ctx, `
				INSERT INTO day_records (user_id, calendar_date)
				VALUES ($1, $2) RETURNING id
			`, userID, event.CalendarDate).Scan(&dayRecordID)
		}
		if err != nil {
			return 0, err
		}
		dayRecordIDs[dayRecordID] = struct{}{}
		commandTag, insertError := transaction.Exec(ctx, `
			INSERT INTO day_events
			(day_record_id, device_id, client_event_id, event_type, category_id, occurred_at, occurred_at_local,
			 target_client_event_id, corrected_at, corrected_at_local)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT DO NOTHING
		`, dayRecordID, nil, event.ClientEventID, event.EventType,
			mappedEventCategoryID(categoryIDs, event.CategoryID), event.OccurredAt, event.OccurredAtLocal,
			event.TargetClientEventID, event.CorrectedAt, event.CorrectedAtLocal)
		if insertError != nil {
			return 0, insertError
		}
		if commandTag.RowsAffected() > 0 {
			importedEvents++
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return 0, err
	}
	return importedEvents, nil
}

func (repository *BackupRepository) insertTemplateSnapshot(ctx context.Context, transaction pgx.Tx, userID, templateID int, plan []BackupPlanBlock, categoryIDs map[int]int) (int, error) {
	var snapshotID int
	if err := transaction.QueryRow(ctx, `INSERT INTO template_snapshots (day_template_id, user_id) VALUES ($1, $2) RETURNING id`, templateID, userID).Scan(&snapshotID); err != nil {
		return 0, err
	}
	for _, block := range plan {
		categoryID, exists := categoryIDs[block.CategoryID]
		if !exists {
			categoryID = block.CategoryID
		}
		if _, err := transaction.Exec(ctx, `INSERT INTO snapshot_blocks (snapshot_id, category_id, start_time, duration_minutes) VALUES ($1, $2, $3, $4)`, snapshotID, categoryID, block.StartTime, block.DurationMinutes); err != nil {
			return 0, err
		}
	}
	return snapshotID, nil
}

func mappedEventCategoryID(categoryIDs map[int]int, categoryID *int) *int {
	if categoryID == nil {
		return nil
	}
	mappedCategoryID, exists := categoryIDs[*categoryID]
	if !exists {
		mappedCategoryID = *categoryID
	}
	return &mappedCategoryID
}
