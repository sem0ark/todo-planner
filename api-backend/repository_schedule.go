package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduleRepository struct {
	db *pgxpool.Pool
}

// WeeklySchedule struct already defined in models.go
type WeeklySchedule struct {
	ID            int        `json:"id"`
	UserID        int        `json:"user_id"`
	DayOfWeek     int        `json:"day_of_week"`
	DayTemplateID *int       `json:"day_template_id"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// ScheduleOverride struct already defined in models.go
type ScheduleOverride struct {
	ID            int        `json:"id"`
	UserID        int        `json:"user_id"`
	CalendarDate  string     `json:"calendar_date"`
	DayTemplateID *int       `json:"day_template_id"`
	CreatedAt     *time.Time `json:"created_at"`
}

// A single day's template assignment
type WeeklyScheduleEntry struct {
	DayOfWeek     int  `json:"day_of_week"`
	DayTemplateID *int `json:"day_template_id"`
}

func NewScheduleRepository(db *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// GetWeeklySchedule returns all 7 days of the week (0-6), inserting missing entries
func (r *ScheduleRepository) GetWeeklySchedule(ctx context.Context, userID int) ([]WeeklySchedule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, day_of_week, day_template_id, updated_at
		FROM weekly_schedule
		WHERE user_id = $1
		ORDER BY day_of_week ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	existing := make(map[int]WeeklySchedule)
	for rows.Next() {
		var ws WeeklySchedule
		if err := rows.Scan(&ws.ID, &ws.UserID, &ws.DayOfWeek, &ws.DayTemplateID, &ws.UpdatedAt); err != nil {
			return nil, err
		}
		existing[ws.DayOfWeek] = ws
	}

	// Ensure all 7 days are present
	result := make([]WeeklySchedule, 7)
	for i := 0; i < 7; i++ {
		if ws, ok := existing[i]; ok {
			result[i] = ws
		} else {
			result[i] = WeeklySchedule{
				UserID:        userID,
				DayOfWeek:     i,
				DayTemplateID: nil,
			}
		}
	}

	return result, nil
}

// ReplaceWeeklySchedule replaces all 7 days of the weekly schedule
func (r *ScheduleRepository) ReplaceWeeklySchedule(ctx context.Context, userID int, entries []WeeklyScheduleEntry) ([]WeeklySchedule, error) {
	// Validate input: must have exactly 7 entries, one per day
	if len(entries) != 7 {
		return nil, fmt.Errorf("exactly 7 days required, got %d", len(entries))
	}

	seen := make(map[int]bool)
	for _, entry := range entries {
		if entry.DayOfWeek < 0 || entry.DayOfWeek > 6 {
			return nil, fmt.Errorf("invalid day_of_week: %d", entry.DayOfWeek)
		}
		if seen[entry.DayOfWeek] {
			return nil, fmt.Errorf("duplicate day_of_week: %d", entry.DayOfWeek)
		}
		seen[entry.DayOfWeek] = true
	}
	for _, entry := range entries {
		if entry.DayTemplateID == nil {
			continue
		}
		if err := r.ensureTemplateBelongsToUser(ctx, userID, *entry.DayTemplateID); err != nil {
			return nil, err
		}
	}

	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Upsert each day
	for _, entry := range entries {
		_, err := tx.Exec(ctx, `
			INSERT INTO weekly_schedule (user_id, day_of_week, day_template_id, updated_at)
			VALUES ($1, $2, $3, now())
			ON CONFLICT (user_id, day_of_week)
			DO UPDATE SET day_template_id = EXCLUDED.day_template_id, updated_at = now()
		`, userID, entry.DayOfWeek, entry.DayTemplateID)
		if err != nil {
			return nil, err
		}
	}
	if err := r.repinActiveAndFutureRecords(ctx, tx, userID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Return updated schedule
	return r.GetWeeklySchedule(ctx, userID)
}

// GetFutureOverrides returns all overrides from today onward
func (r *ScheduleRepository) GetFutureOverrides(ctx context.Context, userID int) ([]ScheduleOverride, error) {
	today := time.Now().Format("2006-01-02")

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, calendar_date::text, day_template_id, created_at
		FROM schedule_overrides
		WHERE user_id = $1 AND calendar_date >= $2
		ORDER BY calendar_date ASC
	`, userID, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	overrides := make([]ScheduleOverride, 0)
	for rows.Next() {
		var so ScheduleOverride
		if err := rows.Scan(&so.ID, &so.UserID, &so.CalendarDate, &so.DayTemplateID, &so.CreatedAt); err != nil {
			return nil, err
		}
		overrides = append(overrides, so)
	}

	return overrides, nil
}

// GetTemplateForDate resolves the template assigned to a calendar date.
// A date-specific override takes precedence over the weekly schedule.
func (r *ScheduleRepository) GetTemplateForDate(ctx context.Context, userID int, calendarDate string) (*int, error) {
	var templateID *int
	err := r.db.QueryRow(ctx, `
		SELECT day_template_id
		FROM schedule_overrides
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&templateID)
	if err == nil {
		return templateID, nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	parsedDate, err := time.Parse("2006-01-02", calendarDate)
	if err != nil {
		return nil, err
	}
	dayOfWeek := (int(parsedDate.Weekday()) + 6) % 7

	err = r.db.QueryRow(ctx, `
		SELECT day_template_id
		FROM weekly_schedule
		WHERE user_id = $1 AND day_of_week = $2
	`, userID, dayOfWeek).Scan(&templateID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return templateID, nil
}

// SetOverride creates or updates a schedule override. If dayTemplateID is nil, removes the override.
func (r *ScheduleRepository) SetOverride(ctx context.Context, userID int, calendarDate string, dayTemplateID *int) (*ScheduleOverride, error) {
	if dayTemplateID != nil {
		if err := r.ensureTemplateBelongsToUser(ctx, userID, *dayTemplateID); err != nil {
			return nil, err
		}
	}
	transaction, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	// If dayTemplateID is nil, delete the override
	if dayTemplateID == nil {
		_, err := transaction.Exec(ctx, `
			DELETE FROM schedule_overrides
			WHERE user_id = $1 AND calendar_date = $2
		`, userID, calendarDate)
		if err != nil {
			return nil, err
		}
	} else {
		_, err = transaction.Exec(ctx, `
			INSERT INTO schedule_overrides (user_id, calendar_date, day_template_id, created_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, calendar_date)
		DO UPDATE SET day_template_id = EXCLUDED.day_template_id
		`, userID, calendarDate, dayTemplateID)
		if err != nil {
			return nil, err
		}
	}
	if err := r.repinDate(ctx, transaction, userID, calendarDate); err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	override, err := r.GetOverride(ctx, userID, calendarDate)
	if err != nil {
		return nil, err
	}
	if override != nil {
		return override, nil
	}

	return &ScheduleOverride{
		UserID:        userID,
		CalendarDate:  calendarDate,
		DayTemplateID: dayTemplateID,
	}, nil
}

func (r *ScheduleRepository) ensureTemplateBelongsToUser(ctx context.Context, userID, templateID int) error {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM day_templates WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE)
	`, templateID, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("day template not found")
	}
	return nil
}

func (r *ScheduleRepository) repinActiveAndFutureRecords(ctx context.Context, transaction pgx.Tx, userID int) error {
	rows, err := transaction.Query(ctx, `
		SELECT id, calendar_date::text
		FROM day_records
		WHERE user_id = $1 AND calendar_date >= CURRENT_DATE
	`, userID)
	if err != nil {
		return err
	}
	type recordDate struct {
		id   int
		date string
	}
	recordDates := make([]recordDate, 0)
	for rows.Next() {
		var record recordDate
		if err := rows.Scan(&record.id, &record.date); err != nil {
			return err
		}
		recordDates = append(recordDates, record)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()
	for _, record := range recordDates {
		if err := r.repinRecord(ctx, transaction, userID, record.id, record.date); err != nil {
			return err
		}
	}
	return nil
}

func (r *ScheduleRepository) repinDate(ctx context.Context, transaction pgx.Tx, userID int, calendarDate string) error {
	var recordID int
	err := transaction.QueryRow(ctx, `
		SELECT id FROM day_records
		WHERE user_id = $1 AND calendar_date = $2 AND calendar_date >= CURRENT_DATE
	`, userID, calendarDate).Scan(&recordID)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	return r.repinRecord(ctx, transaction, userID, recordID, calendarDate)
}

func (r *ScheduleRepository) repinRecord(ctx context.Context, transaction pgx.Tx, userID, recordID int, calendarDate string) error {
	templateID, err := resolveTemplateForDateTx(ctx, transaction, userID, calendarDate)
	if err != nil {
		return err
	}
	var snapshotID *int
	if templateID != nil {
		var currentSnapshotID int
		err = transaction.QueryRow(ctx, `
			SELECT id FROM template_snapshots
			WHERE day_template_id = $1
			ORDER BY snapshotted_at DESC, id DESC LIMIT 1
		`, *templateID).Scan(&currentSnapshotID)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == nil {
			snapshotID = &currentSnapshotID
		}
	}
	_, err = transaction.Exec(ctx, `
		UPDATE day_records
		SET day_template_id = $1, snapshot_id = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4 AND calendar_date >= CURRENT_DATE
	`, templateID, snapshotID, recordID, userID)
	return err
}

func resolveTemplateForDateTx(ctx context.Context, transaction pgx.Tx, userID int, calendarDate string) (*int, error) {
	var templateID *int
	err := transaction.QueryRow(ctx, `
		SELECT day_template_id FROM schedule_overrides
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&templateID)
	if err == nil {
		return templateID, nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}
	parsedDate, err := time.Parse("2006-01-02", calendarDate)
	if err != nil {
		return nil, err
	}
	dayOfWeek := (int(parsedDate.Weekday()) + 6) % 7
	err = transaction.QueryRow(ctx, `
		SELECT day_template_id FROM weekly_schedule
		WHERE user_id = $1 AND day_of_week = $2
	`, userID, dayOfWeek).Scan(&templateID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return templateID, err
}

// GetOverride retrieves a specific override by date
func (r *ScheduleRepository) GetOverride(ctx context.Context, userID int, calendarDate string) (*ScheduleOverride, error) {
	var so ScheduleOverride
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, calendar_date::text, day_template_id, created_at
		FROM schedule_overrides
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&so.ID, &so.UserID, &so.CalendarDate, &so.DayTemplateID, &so.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &so, nil
}
