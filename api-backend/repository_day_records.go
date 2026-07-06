package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DayRecordRepository struct {
	db *pgxpool.Pool
}

func NewDayRecordRepository(db *pgxpool.Pool) *DayRecordRepository {
	return &DayRecordRepository{db: db}
}

// FindByDateRange returns all day records for a user within the specified date range
func (r *DayRecordRepository) FindByDateRange(ctx context.Context, userID int, fromDate, toDate string) ([]DayRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, snapshot_id, calendar_date::text, review_status, created_at, updated_at
		FROM day_records
		WHERE user_id = $1 AND calendar_date >= $2 AND calendar_date <= $3
		ORDER BY calendar_date ASC
	`, userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]DayRecord, 0)
	for rows.Next() {
		var rec DayRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.SnapshotID, &rec.CalendarDate, &rec.ReviewStatus, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}

		// Load snapshot blocks if snapshot exists
		if rec.SnapshotID != nil {
			snapshotBlocks, err := r.getSnapshotBlocks(ctx, *rec.SnapshotID)
			if err != nil {
				return nil, err
			}
			rec.SnapshotBlocks = snapshotBlocks
		}

		// Load actual blocks
		actualBlocks, err := r.getActualBlocks(ctx, rec.ID)
		if err != nil {
			return nil, err
		}
		rec.ActualBlocks = actualBlocks

		records = append(records, rec)
	}

	return records, nil
}

// FindByID returns a single day record by ID for a specific user
func (r *DayRecordRepository) FindByID(ctx context.Context, id, userID int) (*DayRecord, error) {
	var rec DayRecord
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, snapshot_id, calendar_date::text, review_status, created_at, updated_at
		FROM day_records
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&rec.ID, &rec.UserID, &rec.SnapshotID, &rec.CalendarDate, &rec.ReviewStatus, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Load snapshot blocks if snapshot exists
	if rec.SnapshotID != nil {
		snapshotBlocks, err := r.getSnapshotBlocks(ctx, *rec.SnapshotID)
		if err != nil {
			return nil, err
		}
		rec.SnapshotBlocks = snapshotBlocks
	}

	// Load actual blocks
	actualBlocks, err := r.getActualBlocks(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	rec.ActualBlocks = actualBlocks

	return &rec, nil
}

// Create creates a new day record and pins the current template snapshot
func (r *DayRecordRepository) Create(ctx context.Context, userID int, calendarDate string) (*DayRecord, error) {
	// Check if a record already exists for this date
	var existingID int
	err := r.db.QueryRow(ctx, `
		SELECT id FROM day_records WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("day record already exists for date %s", calendarDate)
	}

	// Resolve the active template for this date
	templateID, err := r.resolveTemplateForDate(ctx, userID, calendarDate)
	if err != nil {
		return nil, err
	}

	// Get the most recent snapshot for this template (if template exists)
	var snapshotID *int
	if templateID != nil {
		snapshot, err := r.getLatestSnapshot(ctx, *templateID)
		if err != nil {
			return nil, err
		}
		if snapshot != nil {
			snapshotID = &snapshot.ID
		}
	}

	// Create the day record
	var rec DayRecord
	now := time.Now()
	err = r.db.QueryRow(ctx, `
		INSERT INTO day_records (user_id, snapshot_id, calendar_date, review_status, created_at, updated_at)
		VALUES ($1, $2, $3, 'Unreviewed', $4, $5)
		RETURNING id, user_id, snapshot_id, calendar_date::text, review_status, created_at, updated_at
	`, userID, snapshotID, calendarDate, now, now).Scan(
		&rec.ID, &rec.UserID, &rec.SnapshotID, &rec.CalendarDate, &rec.ReviewStatus, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Load snapshot blocks if snapshot exists
	if rec.SnapshotID != nil {
		snapshotBlocks, err := r.getSnapshotBlocks(ctx, *rec.SnapshotID)
		if err != nil {
			return nil, err
		}
		rec.SnapshotBlocks = snapshotBlocks
	}

	// Actual blocks are initially empty
	rec.ActualBlocks = []ActualBlock{}

	return &rec, nil
}

// UpdateStatus transitions the review status of a day record
func (r *DayRecordRepository) UpdateStatus(ctx context.Context, id, userID int, reviewStatus string) (*DayRecord, error) {
	// Validate status
	if reviewStatus != "Reviewed" && reviewStatus != "Ignored" {
		return nil, fmt.Errorf("invalid review_status: must be 'Reviewed' or 'Ignored'")
	}

	// Check current status
	var currentStatus string
	err := r.db.QueryRow(ctx, `
		SELECT review_status FROM day_records WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&currentStatus)
	if err != nil {
		return nil, err
	}

	// Prevent transitions from Reviewed or Ignored
	if currentStatus == "Reviewed" || currentStatus == "Ignored" {
		return nil, fmt.Errorf("cannot change status: record is already %s", currentStatus)
	}

	// Update status
	now := time.Now()
	var rec DayRecord
	err = r.db.QueryRow(ctx, `
		UPDATE day_records
		SET review_status = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
		RETURNING id, user_id, snapshot_id, calendar_date::text, review_status, created_at, updated_at
	`, reviewStatus, now, id, userID).Scan(
		&rec.ID, &rec.UserID, &rec.SnapshotID, &rec.CalendarDate, &rec.ReviewStatus, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Load snapshot blocks if snapshot exists
	if rec.SnapshotID != nil {
		snapshotBlocks, err := r.getSnapshotBlocks(ctx, *rec.SnapshotID)
		if err != nil {
			return nil, err
		}
		rec.SnapshotBlocks = snapshotBlocks
	}

	// Load actual blocks
	actualBlocks, err := r.getActualBlocks(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	rec.ActualBlocks = actualBlocks

	return &rec, nil
}

// Helper: resolve the active template for a date
func (r *DayRecordRepository) resolveTemplateForDate(ctx context.Context, userID int, calendarDate string) (*int, error) {
	// Check for schedule override first
	var templateID *int
	err := r.db.QueryRow(ctx, `
		SELECT day_template_id FROM schedule_overrides
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&templateID)
	if err == nil {
		return templateID, nil
	}

	// Fall back to weekly schedule
	// Convert date to day of week (0=Monday, 6=Sunday)
	t, err := time.Parse("2006-01-02", calendarDate)
	if err != nil {
		return nil, err
	}
	dayOfWeek := int(t.Weekday())
	// Adjust: Go uses 0=Sunday, we use 0=Monday
	dayOfWeek = (dayOfWeek + 6) % 7

	err = r.db.QueryRow(ctx, `
		SELECT day_template_id FROM weekly_schedule
		WHERE user_id = $1 AND day_of_week = $2
	`, userID, dayOfWeek).Scan(&templateID)
	if err != nil {
		// No weekly schedule entry found - return nil (no template)
		return nil, nil
	}

	return templateID, nil
}

// Helper: get the latest snapshot for a template
func (r *DayRecordRepository) getLatestSnapshot(ctx context.Context, templateID int) (*TemplateSnapshot, error) {
	var snapshot TemplateSnapshot
	err := r.db.QueryRow(ctx, `
		SELECT id, day_template_id, user_id, snapshotted_at
		FROM template_snapshots
		WHERE day_template_id = $1
		ORDER BY snapshotted_at DESC
		LIMIT 1
	`, templateID).Scan(&snapshot.ID, &snapshot.DayTemplateID, &snapshot.UserID, &snapshot.SnapshottedAt)
	if err != nil {
		// No snapshot found
		return nil, nil
	}

	return &snapshot, nil
}

// Helper: get snapshot blocks for a snapshot
func (r *DayRecordRepository) getSnapshotBlocks(ctx context.Context, snapshotID int) ([]SnapshotBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, snapshot_id, category_id, start_time, duration_minutes
		FROM snapshot_blocks
		WHERE snapshot_id = $1
		ORDER BY start_time ASC
	`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]SnapshotBlock, 0)
	for rows.Next() {
		var block SnapshotBlock
		if err := rows.Scan(&block.ID, &block.SnapshotID, &block.CategoryID, &block.StartTime, &block.DurationMinutes); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// Helper: get actual blocks for a day record
func (r *DayRecordRepository) getActualBlocks(ctx context.Context, dayRecordID int) ([]ActualBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, day_record_id, category_id, block_type, start_time, duration_minutes, updated_at
		FROM actual_blocks
		WHERE day_record_id = $1
		ORDER BY start_time ASC
	`, dayRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]ActualBlock, 0)
	for rows.Next() {
		var block ActualBlock
		if err := rows.Scan(&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType, &block.StartTime, &block.DurationMinutes, &block.UpdatedAt); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}
