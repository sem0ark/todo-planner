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

type TemplateSnapshot struct {
	ID             int             `json:"id"`
	DayTemplateID  int             `json:"day_template_id"`
	UserID         int             `json:"user_id"`
	SnapshotBlocks []SnapshotBlock `json:"snapshot_blocks"`
	SnapshottedAt  time.Time       `json:"snapshotted_at"`
}

type SnapshotBlock struct {
	ID              int    `json:"id"`
	SnapshotID      int    `json:"snapshot_id"`
	CategoryID      int    `json:"category_id"`
	StartTime       string `json:"start_time"` // HH:MM:SS
	DurationMinutes int    `json:"duration_minutes"`
}

type DayRecord struct {
	ID             int             `json:"id"`
	UserID         int             `json:"user_id"`
	SnapshotID     *int            `json:"snapshot_id"`
	CalendarDate   string          `json:"calendar_date"` // YYYY-MM-DD
	ReviewStatus   string          `json:"review_status"` // Unreviewed | Reviewed | Ignored
	SnapshotBlocks []SnapshotBlock `json:"snapshot_blocks"`
	ActualBlocks   []ActualBlock   `json:"actual_blocks"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type DayEventInput struct {
	EventType          string    `json:"event_type"` // confirmation | transition
	OutgoingCategoryID *int      `json:"outgoing_category_id"`
	IncomingCategoryID *int      `json:"incoming_category_id"`
	OccurredAt         time.Time `json:"occurred_at"`
}

type DayEvent struct {
	ID                 int       `json:"id"`
	DayRecordID        int       `json:"day_record_id"`
	EventType          string    `json:"event_type"` // confirmation | transition
	OutgoingCategoryID *int      `json:"outgoing_category_id"`
	IncomingCategoryID *int      `json:"incoming_category_id"`
	OccurredAt         time.Time `json:"occurred_at"`
}

type ActualBlock struct {
	ID              int       `json:"id"`
	DayRecordID     int       `json:"day_record_id"`
	CategoryID      *int      `json:"category_id"`
	BlockType       string    `json:"block_type"` // actual | blank | untracked
	StartTime       string    `json:"start_time"` // HH:MM:SS
	DurationMinutes int       `json:"duration_minutes"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func NewDayRecordRepository(db *pgxpool.Pool) *DayRecordRepository {
	return &DayRecordRepository{db: db}
}

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
		} else {
			// Initialize empty slice when no snapshot exists
			rec.SnapshotBlocks = make([]SnapshotBlock, 0)
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
	} else {
		// Initialize empty slice when no snapshot exists
		rec.SnapshotBlocks = make([]SnapshotBlock, 0)
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

// CreateEvents appends day events and recomputes actual blocks
func (r *DayRecordRepository) CreateEvents(ctx context.Context, dayRecordID, userID int, inputs []DayEventInput) ([]DayEvent, []ActualBlock, error) {
	// Verify record exists and belongs to user
	var reviewStatus string
	err := r.db.QueryRow(ctx, `
		SELECT review_status FROM day_records WHERE id = $1 AND user_id = $2
	`, dayRecordID, userID).Scan(&reviewStatus)
	if err != nil {
		return nil, nil, err
	}

	// Check review status
	if reviewStatus != "Unreviewed" {
		return nil, nil, fmt.Errorf("cannot add events: day record is %s", reviewStatus)
	}

	// Insert events
	createdEvents := make([]DayEvent, 0, len(inputs))
	for _, input := range inputs {
		var event DayEvent
		err := r.db.QueryRow(ctx, `
			INSERT INTO day_events (day_record_id, event_type, outgoing_category_id, incoming_category_id, occurred_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, day_record_id, event_type, outgoing_category_id, incoming_category_id, occurred_at
		`, dayRecordID, input.EventType, input.OutgoingCategoryID, input.IncomingCategoryID, input.OccurredAt).Scan(
			&event.ID, &event.DayRecordID, &event.EventType, &event.OutgoingCategoryID, &event.IncomingCategoryID, &event.OccurredAt,
		)
		if err != nil {
			return nil, nil, err
		}
		createdEvents = append(createdEvents, event)
	}

	// Recompute actual blocks
	actualBlocks, err := r.recomputeActualBlocks(ctx, dayRecordID)
	if err != nil {
		return nil, nil, err
	}

	return createdEvents, actualBlocks, nil
}

func (r *DayRecordRepository) recomputeActualBlocks(ctx context.Context, dayRecordID int) ([]ActualBlock, error) {
	events, err := r.getDayEvents(ctx, dayRecordID)
	if err != nil {
		return nil, err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM actual_blocks WHERE day_record_id = $1`, dayRecordID)
	if err != nil {
		return nil, err
	}

	computedBlocks := computeActualBlocks(events, time.Now())

	if len(computedBlocks) == 0 {
		return []ActualBlock{}, nil
	}

	// Persist computed blocks to database
	blocks := make([]ActualBlock, 0, len(computedBlocks))
	now := time.Now()

	for _, computed := range computedBlocks {
		var block ActualBlock
		err := r.db.QueryRow(ctx, `
			INSERT INTO actual_blocks (day_record_id, category_id, block_type, start_time, duration_minutes, updated_at)
			VALUES ($1, $2, 'actual', $3, $4, $5)
			RETURNING id, day_record_id, category_id, block_type, start_time, duration_minutes, updated_at
		`, dayRecordID, computed.CategoryID, computed.StartTime.Format("15:04:05"), computed.DurationMinutes, now).Scan(
			&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType, &block.StartTime, &block.DurationMinutes, &block.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (r *DayRecordRepository) getDayEvents(ctx context.Context, dayRecordID int) ([]DayEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, day_record_id, event_type, outgoing_category_id, incoming_category_id, occurred_at
		FROM day_events
		WHERE day_record_id = $1
		ORDER BY occurred_at ASC
	`, dayRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]DayEvent, 0)
	for rows.Next() {
		var event DayEvent
		if err := rows.Scan(&event.ID, &event.DayRecordID, &event.EventType, &event.OutgoingCategoryID, &event.IncomingCategoryID, &event.OccurredAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

type ComputedBlock struct {
	CategoryID      *int
	StartTime       time.Time
	DurationMinutes int
}

func computeActualBlocks(events []DayEvent, referenceTime time.Time) []ComputedBlock {
	if len(events) == 0 {
		return []ComputedBlock{}
	}

	computedBlocks := make([]ComputedBlock, 0)

	for i := 0; i < len(events); i++ {
		event := events[i]

		// Skip confirmation events for block computation
		if event.EventType == "confirmation" {
			continue
		}

		// For transition events, create a block
		if event.EventType == "transition" {
			startTime := event.OccurredAt
			categoryID := event.IncomingCategoryID

			// Find next transition to determine end time
			var endTime time.Time
			if i+1 < len(events) {
				for j := i + 1; j < len(events); j++ {
					if events[j].EventType == "transition" {
						endTime = events[j].OccurredAt
						break
					}
				}
				// If no next transition found, block extends to referenceTime (ongoing)
				if endTime.IsZero() {
					endTime = referenceTime
				}
			} else {
				// Last event, block extends to referenceTime
				endTime = referenceTime
			}

			// Calculate duration in minutes
			duration := int(endTime.Sub(startTime).Minutes())
			if duration <= 0 {
				continue
			}

			// Create computed block
			block := ComputedBlock{
				CategoryID:      categoryID,
				StartTime:       startTime,
				DurationMinutes: duration,
			}
			computedBlocks = append(computedBlocks, block)
		}
	}

	return computedBlocks
}
