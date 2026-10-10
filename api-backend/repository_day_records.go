package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
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
	ID              int          `json:"id"`
	SnapshotID      int          `json:"snapshot_id"`
	CategoryID      int          `json:"category_id"`
	StartTime       ScheduleTime `json:"-"`
	DurationMinutes int          `json:"duration_minutes"`
}

type DayRecord struct {
	ID                    int             `json:"id"`
	UserID                int             `json:"user_id"`
	SnapshotID            *int            `json:"snapshot_id"`
	DayTemplateID         *int            `json:"day_template_id"`
	CalendarDate          CalendarDate    `json:"calendar_date"`
	TimezoneOffsetMinutes *int            `json:"timezone_offset_minutes"`
	TimezoneOffsetLocked  bool            `json:"timezone_offset_locked"`
	SnapshotBlocks        []SnapshotBlock `json:"snapshot_blocks"`
	ActualBlocks          []ActualBlock   `json:"actual_blocks"`
	Events                []DayEvent      `json:"events"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	SnapshottedAt         time.Time       `json:"-"`
}

type DayRangeEntry struct {
	CalendarDate CalendarDate
	Record       *DayRecord
}

type DayEventInput struct {
	ClientEventID       string     `json:"client_event_id"`
	EventType           string     `json:"event_type"` // confirmation | transition | note | amendment
	CategoryID          *int       `json:"category_id"`
	Text                *string    `json:"text"`
	OccurredAt          time.Time  `json:"occurred_at"`
	OccurredAtLocal     string     `json:"occurred_at_local"`
	TargetClientEventID string     `json:"target_client_event_id"`
	CorrectedAt         *time.Time `json:"corrected_at"`
	CorrectedAtLocal    *string    `json:"corrected_at_local,omitempty"`
}

type DayEventsInput struct {
	DeviceID int             `json:"device_id"`
	Events   []DayEventInput `json:"events"`
}

type DayEvent struct {
	ID                  int        `json:"id"`
	DayRecordID         int        `json:"day_record_id"`
	EventType           string     `json:"event_type"` // confirmation | transition | note
	CategoryID          *int       `json:"category_id"`
	Text                *string    `json:"text,omitempty"`
	OccurredAt          time.Time  `json:"occurred_at"`
	OccurredAtLocal     string     `json:"occurred_at_local"`
	ClientEventID       *string    `json:"client_event_id,omitempty"`
	TargetClientEventID *string    `json:"target_client_event_id,omitempty"`
	CorrectedAt         *time.Time `json:"corrected_at,omitempty"`
	CorrectedAtLocal    *string    `json:"corrected_at_local,omitempty"`
	DeviceID            *int       `json:"device_id,omitempty"`
}

type ActualBlock struct {
	ID              int          `json:"id"`
	DayRecordID     int          `json:"day_record_id"`
	CategoryID      *int         `json:"category_id"`
	BlockType       string       `json:"block_type"` // actual | blank | untracked
	StartTime       ScheduleTime `json:"-"`
	DurationMinutes int          `json:"duration_minutes"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type DateEventResult struct {
	Record            *DayRecord
	AcceptedEvents    []DayEvent
	DuplicateEventIDs []string
}

var (
	ErrDayRecordPast           = NewBadRequestError("day record is in the past")
	ErrDayRecordNotFound       = NewAppErrorWithCause(http.StatusNotFound, "day record not found", pgx.ErrNoRows)
	ErrDayRecordAlreadyExists  = NewConflictError("day record already exists")
	ErrDeviceNotFound          = NewNotFoundError("device not found")
	ErrAmendmentTargetNotFound = NewBadRequestError("amendment target event not found")
)

func NewDayRecordRepository(db *pgxpool.Pool) *DayRecordRepository {
	return &DayRecordRepository{db: db}
}

func (r *DayRecordRepository) FindByDateRange(ctx context.Context, userID int, fromDate, toDate CalendarDate) ([]DayRecord, error) {
	batch := newPgBatchBuilder()
	batch.Queue(`
		SELECT records.id, records.user_id, records.day_template_id, records.snapshot_id,
			records.calendar_date, records.timezone_offset_minutes,
			records.timezone_offset_locked, records.created_at, records.updated_at,
			snapshots.snapshotted_at
		FROM day_records AS records
		LEFT JOIN template_snapshots AS snapshots ON snapshots.id = records.snapshot_id
		WHERE records.user_id = $1 AND records.calendar_date >= $2 AND records.calendar_date <= $3
		ORDER BY records.calendar_date ASC
	`, userID, fromDate, toDate)
	batch.Queue(`
		SELECT records.id, blocks.id, blocks.snapshot_id, blocks.category_id,
			blocks.start_time, blocks.duration_minutes
		FROM day_records AS records
		JOIN snapshot_blocks AS blocks ON blocks.snapshot_id = records.snapshot_id
		WHERE records.user_id = $1 AND records.calendar_date >= $2 AND records.calendar_date <= $3
		ORDER BY records.id, blocks.start_time ASC
	`, userID, fromDate, toDate)
	batch.Queue(`
		SELECT day_record_id, id, category_id, block_type, start_time, duration_minutes, updated_at
		FROM actual_blocks
		WHERE day_record_id IN (
			SELECT id FROM day_records
			WHERE user_id = $1 AND calendar_date >= $2 AND calendar_date <= $3
		)
		ORDER BY day_record_id, start_time ASC
	`, userID, fromDate, toDate)
	batch.Queue(`
		SELECT day_record_id, id, device_id, client_event_id, event_type, category_id, text,
			occurred_at, COALESCE(occurred_at_local, ''), target_client_event_id,
			corrected_at, corrected_at_local
		FROM day_events
		WHERE day_record_id IN (
			SELECT id FROM day_records
			WHERE user_id = $1 AND calendar_date >= $2 AND calendar_date <= $3
		)
		ORDER BY day_record_id, id ASC
	`, userID, fromDate, toDate)

	var records []DayRecord
	if err := batch.Execute(ctx, r.db, func(batchResults pgx.BatchResults) error {
		recordIndexByID := make(map[int]int)
		recordRows, err := batchResults.Query()
		if err != nil {
			return err
		}
		for recordRows.Next() {
			var record DayRecord
			var snapshottedAt *time.Time
			if err := recordRows.Scan(
				&record.ID, &record.UserID, &record.DayTemplateID, &record.SnapshotID,
				&record.CalendarDate, &record.TimezoneOffsetMinutes,
				&record.TimezoneOffsetLocked, &record.CreatedAt, &record.UpdatedAt,
				&snapshottedAt,
			); err != nil {
				recordRows.Close()
				return err
			}
			if snapshottedAt != nil {
				record.SnapshottedAt = *snapshottedAt
			}
			record.SnapshotBlocks = []SnapshotBlock{}
			record.ActualBlocks = []ActualBlock{}
			record.Events = []DayEvent{}
			recordIndexByID[record.ID] = len(records)
			records = append(records, record)
		}
		if err := recordRows.Err(); err != nil {
			recordRows.Close()
			return err
		}
		recordRows.Close()

		snapshotRows, err := batchResults.Query()
		if err != nil {
			return err
		}
		for snapshotRows.Next() {
			var recordID int
			var block SnapshotBlock
			if err := snapshotRows.Scan(&recordID, &block.ID, &block.SnapshotID, &block.CategoryID, &block.StartTime, &block.DurationMinutes); err != nil {
				snapshotRows.Close()
				return err
			}
			records[recordIndexByID[recordID]].SnapshotBlocks = append(records[recordIndexByID[recordID]].SnapshotBlocks, block)
		}
		if err := snapshotRows.Err(); err != nil {
			snapshotRows.Close()
			return err
		}
		snapshotRows.Close()

		actualRows, err := batchResults.Query()
		if err != nil {
			return err
		}
		for actualRows.Next() {
			var recordID int
			var block ActualBlock
			if err := actualRows.Scan(&recordID, &block.ID, &block.CategoryID, &block.BlockType, &block.StartTime, &block.DurationMinutes, &block.UpdatedAt); err != nil {
				actualRows.Close()
				return err
			}
			block.DayRecordID = recordID
			records[recordIndexByID[recordID]].ActualBlocks = append(records[recordIndexByID[recordID]].ActualBlocks, block)
		}
		if err := actualRows.Err(); err != nil {
			actualRows.Close()
			return err
		}
		actualRows.Close()

		eventRows, err := batchResults.Query()
		if err != nil {
			return err
		}
		for eventRows.Next() {
			var event DayEvent
			if err := eventRows.Scan(&event.DayRecordID, &event.ID, &event.DeviceID, &event.ClientEventID, &event.EventType, &event.CategoryID, &event.Text, &event.OccurredAt, &event.OccurredAtLocal, &event.TargetClientEventID, &event.CorrectedAt, &event.CorrectedAtLocal); err != nil {
				eventRows.Close()
				return err
			}
			records[recordIndexByID[event.DayRecordID]].Events = append(records[recordIndexByID[event.DayRecordID]].Events, event)
		}
		return eventRows.Err()
	}); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *DayRecordRepository) FindDateRange(ctx context.Context, userID int, fromDate, toDate CalendarDate) ([]DayRangeEntry, error) {
	records, err := r.FindByDateRange(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	recordsByDate := make(map[CalendarDate]*DayRecord, len(records))
	for recordIndex := range records {
		record := &records[recordIndex]
		recordsByDate[record.CalendarDate] = record
	}

	dateCount := int(toDate.Sub(fromDate).Hours()/24) + 1
	entries := make([]DayRangeEntry, 0, dateCount)
	for currentDate := fromDate; !currentDate.After(toDate); currentDate = currentDate.AddDate(0, 0, 1) {
		entries = append(entries, DayRangeEntry{CalendarDate: currentDate, Record: recordsByDate[currentDate]})
	}
	return entries, nil
}

func (r *DayRecordRepository) FindByDate(ctx context.Context, userID int, calendarDate CalendarDate) (*DayRecord, error) {
	var record DayRecord
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, day_template_id, snapshot_id, calendar_date,
			timezone_offset_minutes, timezone_offset_locked, created_at, updated_at
		FROM day_records
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&record.ID, &record.UserID, &record.DayTemplateID, &record.SnapshotID, &record.CalendarDate, &record.TimezoneOffsetMinutes, &record.TimezoneOffsetLocked, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDayRecordNotFound
		}
		return nil, err
	}
	return r.populateRecord(ctx, &record)
}

func (r *DayRecordRepository) populateRecord(ctx context.Context, record *DayRecord) (*DayRecord, error) {
	if record.SnapshotID != nil {
		record.SnapshotBlocks = make([]SnapshotBlock, 0)
	} else {
		record.SnapshotBlocks = []SnapshotBlock{}
	}
	if err := r.populateRecordCollections(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (r *DayRecordRepository) populateRecordCollections(ctx context.Context, record *DayRecord) error {
	batch := newPgBatchBuilder()
	if record.SnapshotID != nil {
		batch.Queue(`SELECT snapshotted_at FROM template_snapshots WHERE id = $1`, *record.SnapshotID)
		batch.Queue(`
			SELECT id, snapshot_id, category_id, start_time, duration_minutes
			FROM snapshot_blocks
			WHERE snapshot_id = $1
			ORDER BY start_time ASC
		`, *record.SnapshotID)
	}
	batch.Queue(`
		SELECT id, day_record_id, category_id, block_type, start_time, duration_minutes, updated_at
		FROM actual_blocks
		WHERE day_record_id = $1
		ORDER BY start_time ASC
	`, record.ID)
	batch.Queue(`
		SELECT id, day_record_id, device_id, client_event_id, event_type, category_id, text,
			occurred_at, COALESCE(occurred_at_local, ''), target_client_event_id,
			corrected_at, corrected_at_local
		FROM day_events
		WHERE day_record_id = $1
		ORDER BY id ASC
	`, record.ID)

	return batch.Execute(ctx, r.db, func(batchResults pgx.BatchResults) error {
		if record.SnapshotID != nil {
			if err := batchResults.QueryRow().Scan(&record.SnapshottedAt); err != nil {
				return err
			}
			snapshotBlocks, err := scanSnapshotBlocks(batchResults)
			if err != nil {
				return err
			}
			record.SnapshotBlocks = snapshotBlocks
		}
		actualBlocks, err := scanActualBlocks(batchResults)
		if err != nil {
			return err
		}
		record.ActualBlocks = actualBlocks
		events, err := scanDayEvents(batchResults)
		if err != nil {
			return err
		}
		record.Events = events
		return nil
	})
}

func scanSnapshotBlocks(batchResults pgx.BatchResults) ([]SnapshotBlock, error) {
	rows, err := batchResults.Query()
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
	return blocks, rows.Err()
}

func scanActualBlocks(batchResults pgx.BatchResults) ([]ActualBlock, error) {
	rows, err := batchResults.Query()
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
	return blocks, rows.Err()
}

func scanDayEvents(batchResults pgx.BatchResults) ([]DayEvent, error) {
	rows, err := batchResults.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]DayEvent, 0)
	for rows.Next() {
		var event DayEvent
		if err := rows.Scan(&event.ID, &event.DayRecordID, &event.DeviceID, &event.ClientEventID, &event.EventType, &event.CategoryID, &event.Text, &event.OccurredAt, &event.OccurredAtLocal, &event.TargetClientEventID, &event.CorrectedAt, &event.CorrectedAtLocal); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// FindByID returns a single day record by ID for a specific user
func (r *DayRecordRepository) FindByID(ctx context.Context, id, userID int) (*DayRecord, error) {
	var rec DayRecord
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, day_template_id, snapshot_id, calendar_date,
			timezone_offset_minutes, timezone_offset_locked, created_at, updated_at
		FROM day_records
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&rec.ID, &rec.UserID, &rec.DayTemplateID, &rec.SnapshotID, &rec.CalendarDate, &rec.TimezoneOffsetMinutes, &rec.TimezoneOffsetLocked, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return r.populateRecord(ctx, &rec)
}

// Create creates a new day record and pins the current template snapshot
func (r *DayRecordRepository) Create(ctx context.Context, userID int, calendarDate CalendarDate) (*DayRecord, error) {
	// Check if a record already exists for this date
	var existingID int
	err := r.db.QueryRow(ctx, `
		SELECT id FROM day_records WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("%w for date %s", ErrDayRecordAlreadyExists, calendarDate.Format(DateFormat))
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
		INSERT INTO day_records (user_id, day_template_id, snapshot_id, calendar_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, day_template_id, snapshot_id, calendar_date,
			timezone_offset_minutes, timezone_offset_locked, created_at, updated_at
	`, userID, templateID, snapshotID, calendarDate, now, now).Scan(
		&rec.ID, &rec.UserID, &rec.DayTemplateID, &rec.SnapshotID, &rec.CalendarDate,
		&rec.TimezoneOffsetMinutes, &rec.TimezoneOffsetLocked, &rec.CreatedAt, &rec.UpdatedAt,
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
	return r.populateRecord(ctx, &rec)
}

// UpdateTemplate changes only the pinned plan for an active or future record.
func (r *DayRecordRepository) UpdateTemplate(ctx context.Context, userID, recordID int, requestedTemplateID *int) (*DayRecord, error) {
	record, err := r.FindByID(ctx, recordID, userID)
	if err != nil {
		return nil, err
	}
	return r.UpdateTemplateByDate(ctx, userID, record.CalendarDate, requestedTemplateID)
}

// UpdateTemplateByDate is retained for schedule-based internal operations.
func (r *DayRecordRepository) UpdateTemplateByDate(ctx context.Context, userID int, calendarDate CalendarDate, requestedTemplateID *int) (*DayRecord, error) {
	if requestedTemplateID != nil {
		var exists bool
		if err := r.db.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM day_templates WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE)
		`, *requestedTemplateID, userID).Scan(&exists); err != nil {
			return nil, err
		} else if !exists {
			return nil, ErrDayTemplateNotFound
		}
	}

	transaction, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	templateID := requestedTemplateID
	if templateID == nil {
		templateID, err = r.resolveTemplateForDate(ctx, userID, calendarDate)
		if err != nil {
			return nil, err
		}
	}
	var snapshotID *int
	if templateID != nil {
		var currentSnapshotID int
		err = transaction.QueryRow(ctx, `
			SELECT id FROM template_snapshots
			WHERE day_template_id = $1
			ORDER BY snapshotted_at DESC, id DESC LIMIT 1
		`, *templateID).Scan(&currentSnapshotID)
		if err != nil {
			return nil, err
		}
		snapshotID = &currentSnapshotID
	}

	var recordID int
	err = transaction.QueryRow(ctx, `
		UPDATE day_records
		SET day_template_id = $1, snapshot_id = $2, updated_at = NOW()
		WHERE user_id = $3 AND calendar_date = $4 AND calendar_date >= CURRENT_DATE
		RETURNING id
	`, templateID, snapshotID, userID, calendarDate).Scan(&recordID)
	if err == pgx.ErrNoRows {
		var exists bool
		if checkError := transaction.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM day_records WHERE user_id = $1 AND calendar_date = $2)
		`, userID, calendarDate).Scan(&exists); checkError != nil {
			return nil, checkError
		} else if !exists {
			return nil, ErrDayRecordNotFound
		}
		return nil, ErrDayRecordPast
	}
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, recordID, userID)
}

// Helper: resolve the active template for a date
func (r *DayRecordRepository) resolveTemplateForDate(ctx context.Context, userID int, calendarDate CalendarDate) (*int, error) {
	// Check for schedule override first
	var templateID *int
	err := r.db.QueryRow(ctx, `
		SELECT day_template_id FROM schedule_overrides
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&templateID)
	if err == nil {
		return templateID, nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	// Fall back to weekly schedule
	// Convert date to day of week (0=Monday, 6=Sunday)
	dayOfWeek := int(calendarDate.Weekday())
	// Adjust: Go uses 0=Sunday, we use 0=Monday
	dayOfWeek = (dayOfWeek + 6) % 7

	err = r.db.QueryRow(ctx, `
		SELECT day_template_id FROM weekly_schedule
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
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
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

	return blocks, rows.Err()
}

// CreateEvents appends day events and recomputes actual blocks
func (r *DayRecordRepository) CreateEvents(ctx context.Context, dayRecordID, userID int, inputs []DayEventInput) ([]DayEvent, []ActualBlock, error) {
	transaction, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer transaction.Rollback(ctx)

	// Verify record exists and belongs to user
	err = transaction.QueryRow(ctx, `
		SELECT id FROM day_records WHERE id = $1 AND user_id = $2
	`, dayRecordID, userID).Scan(&dayRecordID)
	if err != nil {
		return nil, nil, err
	}

	// Insert events
	createdEvents := make([]DayEvent, 0, len(inputs))
	for _, input := range inputs {
		var event DayEvent
		err := transaction.QueryRow(ctx, `
		INSERT INTO day_events (day_record_id, event_type, category_id, text, occurred_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, day_record_id, event_type, category_id, text, occurred_at
	`, dayRecordID, input.EventType, input.CategoryID, input.Text, input.OccurredAt).Scan(
			&event.ID, &event.DayRecordID, &event.EventType, &event.CategoryID, &event.Text, &event.OccurredAt,
		)
		if err != nil {
			return nil, nil, err
		}
		createdEvents = append(createdEvents, event)
	}

	// Recompute actual blocks
	actualBlocks, _, err := r.recomputeActualBlocks(ctx, transaction, dayRecordID)
	if err != nil {
		return nil, nil, err
	}
	if _, err := transaction.Exec(ctx, `UPDATE day_records SET updated_at = NOW() WHERE id = $1 AND user_id = $2`, dayRecordID, userID); err != nil {
		return nil, nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return createdEvents, actualBlocks, nil
}

// CreateEventsByDate creates the date record when needed and persists a retry-safe batch.
func (r *DayRecordRepository) CreateEventsByDate(ctx context.Context, userID int, calendarDate CalendarDate, deviceID int, inputs []DayEventInput) (*DateEventResult, error) {
	transaction, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	deviceUserID, existingDayRecordID, timezoneOffsetLocked, err := loadEventRequestContext(
		ctx,
		transaction,
		userID,
		calendarDate,
		deviceID,
	)
	if err != nil {
		return nil, err
	}
	if deviceUserID != userID {
		return nil, ErrDeviceNotFound
	}

	dayRecordID, err := findOrCreateDayRecord(
		ctx,
		transaction,
		userID,
		calendarDate,
		existingDayRecordID,
	)
	if err != nil {
		return nil, err
	}
	if err := lockOffsetIfUnset(ctx, transaction, dayRecordID, timezoneOffsetLocked, inputs); err != nil {
		return nil, err
	}
	result := &DateEventResult{
		AcceptedEvents:    make([]DayEvent, 0, len(inputs)),
		DuplicateEventIDs: make([]string, 0),
	}
	lookupEventIDs := eventLookupIDs(inputs)
	existingLocalTimestampsByID, err := getExistingEventLocalTimestamps(
		ctx,
		transaction,
		dayRecordID,
		lookupEventIDs,
	)
	if err != nil {
		return nil, err
	}
	newInputs, duplicateEventIDs, err := prepareNewDateEventInputs(
		inputs,
		existingLocalTimestampsByID,
	)
	if err != nil {
		return nil, err
	}
	result.DuplicateEventIDs = duplicateEventIDs
	var insertedDuplicateEventIDs []string
	result.AcceptedEvents, insertedDuplicateEventIDs, err = insertDateEvents(ctx, transaction, dayRecordID, deviceID, newInputs)
	if err != nil {
		return nil, err
	}
	result.DuplicateEventIDs = append(result.DuplicateEventIDs, insertedDuplicateEventIDs...)
	actualBlocks, events, err := r.recomputeActualBlocks(ctx, transaction, dayRecordID)
	if err != nil {
		return nil, err
	}
	result.Record, err = loadDateEventRecord(ctx, transaction, dayRecordID, userID, actualBlocks, events)
	if err != nil {
		return nil, err
	}
	if err = transaction.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func loadEventRequestContext(
	ctx context.Context,
	transaction pgx.Tx,
	userID int,
	calendarDate CalendarDate,
	deviceID int,
) (int, int, bool, error) {
	batch := newPgBatchBuilder()
	batch.Queue(`SELECT user_id FROM devices WHERE id = $1`, deviceID)
	batch.Queue(`
		SELECT id, timezone_offset_locked
		FROM day_records
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate)

	var deviceUserID int
	var dayRecordID int
	var timezoneOffsetLocked bool
	if err := batch.Execute(ctx, transaction, func(batchResults pgx.BatchResults) error {
		if err := batchResults.QueryRow().Scan(&deviceUserID); err != nil {
			return ErrDeviceNotFound
		}
		if err := batchResults.QueryRow().Scan(&dayRecordID, &timezoneOffsetLocked); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return nil
	}); err != nil {
		return 0, 0, false, err
	}
	return deviceUserID, dayRecordID, timezoneOffsetLocked, nil
}

func loadDateEventRecord(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
	userID int,
	actualBlocks []ActualBlock,
	events []DayEvent,
) (*DayRecord, error) {
	record := &DayRecord{}
	batch := newPgBatchBuilder()
	batch.Queue(`
		WITH updated_record AS (
			UPDATE day_records
			SET updated_at = now()
			WHERE id = $1 AND user_id = $2
			RETURNING id, user_id, day_template_id, snapshot_id, calendar_date,
				timezone_offset_minutes, timezone_offset_locked, created_at, updated_at
		)
		SELECT records.id, records.user_id, records.day_template_id, records.snapshot_id,
			records.calendar_date, records.timezone_offset_minutes,
			records.timezone_offset_locked, records.created_at, records.updated_at,
			snapshots.snapshotted_at
		FROM updated_record AS records
		LEFT JOIN template_snapshots AS snapshots ON snapshots.id = records.snapshot_id
	`, dayRecordID, userID)
	batch.Queue(`
		SELECT blocks.id, blocks.snapshot_id, blocks.category_id,
			blocks.start_time, blocks.duration_minutes
		FROM snapshot_blocks AS blocks
		JOIN day_records AS records ON records.snapshot_id = blocks.snapshot_id
		WHERE records.id = $1 AND records.user_id = $2
		ORDER BY blocks.start_time ASC
	`, dayRecordID, userID)

	err := batch.Execute(ctx, transaction, func(batchResults pgx.BatchResults) error {
		var snapshottedAt *time.Time
		if err := batchResults.QueryRow().Scan(
			&record.ID,
			&record.UserID,
			&record.DayTemplateID,
			&record.SnapshotID,
			&record.CalendarDate,
			&record.TimezoneOffsetMinutes,
			&record.TimezoneOffsetLocked,
			&record.CreatedAt,
			&record.UpdatedAt,
			&snapshottedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrDayRecordNotFound
			}
			return err
		}
		if snapshottedAt != nil {
			record.SnapshottedAt = *snapshottedAt
		}
		snapshotBlocks, err := scanSnapshotBlocks(batchResults)
		if err != nil {
			return err
		}
		record.SnapshotBlocks = snapshotBlocks
		return nil
	})
	if err != nil {
		return nil, err
	}
	record.ActualBlocks = actualBlocks
	record.Events = events
	return record, nil
}

func eventLookupIDs(inputs []DayEventInput) []string {
	lookupIDs := make([]string, 0, len(inputs))
	seenIDs := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if _, alreadySeen := seenIDs[input.ClientEventID]; !alreadySeen {
			seenIDs[input.ClientEventID] = struct{}{}
			lookupIDs = append(lookupIDs, input.ClientEventID)
		}
		if input.EventType != "amendment" {
			continue
		}
		if _, alreadySeen := seenIDs[input.TargetClientEventID]; alreadySeen {
			continue
		}
		seenIDs[input.TargetClientEventID] = struct{}{}
		lookupIDs = append(lookupIDs, input.TargetClientEventID)
	}
	return lookupIDs
}

func getExistingEventLocalTimestamps(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
	clientEventIDs []string,
) (map[string]string, error) {
	existingTimestamps := make(map[string]string, len(clientEventIDs))
	if len(clientEventIDs) == 0 {
		return existingTimestamps, nil
	}

	rows, err := transaction.Query(ctx, `
		SELECT client_event_id, COALESCE(occurred_at_local, '')
		FROM day_events
		WHERE day_record_id = $1 AND client_event_id = ANY($2)
	`, dayRecordID, clientEventIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var clientEventID string
		var occurredAtLocal string
		if err := rows.Scan(&clientEventID, &occurredAtLocal); err != nil {
			return nil, err
		}
		existingTimestamps[clientEventID] = occurredAtLocal
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return existingTimestamps, nil
}

func prepareNewDateEventInputs(
	inputs []DayEventInput,
	existingLocalTimestampsByID map[string]string,
) ([]DayEventInput, []string, error) {
	batchEventIDs := make(map[string]struct{}, len(inputs))
	localTimestampsByID := make(map[string]string, len(inputs))
	processedClientEventIDs := make(map[string]struct{}, len(existingLocalTimestampsByID)+len(inputs))
	for clientEventID := range existingLocalTimestampsByID {
		processedClientEventIDs[clientEventID] = struct{}{}
	}
	for _, input := range inputs {
		batchEventIDs[input.ClientEventID] = struct{}{}
		localTimestampsByID[input.ClientEventID] = input.OccurredAtLocal
	}

	newInputs := make([]DayEventInput, 0, len(inputs))
	duplicateEventIDs := make([]string, 0)
	for _, input := range inputs {
		if _, alreadyProcessed := processedClientEventIDs[input.ClientEventID]; alreadyProcessed {
			duplicateEventIDs = append(duplicateEventIDs, input.ClientEventID)
			continue
		}
		if err := validateDateEventAmendment(
			input,
			batchEventIDs,
			localTimestampsByID,
			existingLocalTimestampsByID,
		); err != nil {
			return nil, nil, err
		}
		newInputs = append(newInputs, input)
		processedClientEventIDs[input.ClientEventID] = struct{}{}
	}
	return newInputs, duplicateEventIDs, nil
}

func validateDateEventAmendment(
	input DayEventInput,
	batchEventIDs map[string]struct{},
	localTimestampsByID map[string]string,
	existingLocalTimestampsByID map[string]string,
) error {
	if input.EventType != "amendment" {
		return nil
	}
	_, targetInBatch := batchEventIDs[input.TargetClientEventID]
	targetLocal := localTimestampsByID[input.TargetClientEventID]
	if !targetInBatch {
		var targetExists bool
		targetLocal, targetExists = existingLocalTimestampsByID[input.TargetClientEventID]
		if !targetExists {
			return ErrAmendmentTargetNotFound
		}
	}
	if input.CorrectedAtLocal == nil {
		return ErrIncompleteAmendment
	}
	return validateAmendmentOffset(targetLocal, *input.CorrectedAtLocal)
}

func insertDateEvents(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
	deviceID int,
	inputs []DayEventInput,
) ([]DayEvent, []string, error) {
	createdEvents := make([]DayEvent, 0, len(inputs))
	duplicateEventIDs := make([]string, 0)
	if len(inputs) == 0 {
		return createdEvents, duplicateEventIDs, nil
	}

	batch := newPgBatchBuilder()
	for _, input := range inputs {
		batch.Queue(`
			INSERT INTO day_events
				(day_record_id, device_id, client_event_id, event_type, category_id, text,
				 occurred_at, occurred_at_local, target_client_event_id, corrected_at,
				 corrected_at_local)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (day_record_id, client_event_id) WHERE client_event_id IS NOT NULL DO NOTHING
			RETURNING id, day_record_id, device_id, client_event_id, event_type, category_id,
				text, occurred_at, occurred_at_local, target_client_event_id, corrected_at,
				corrected_at_local
		`,
			dayRecordID,
			deviceID,
			input.ClientEventID,
			input.EventType,
			input.CategoryID,
			input.Text,
			input.OccurredAt,
			input.OccurredAtLocal,
			nullableEventReference(input.TargetClientEventID),
			input.CorrectedAt,
			input.CorrectedAtLocal,
		)
	}

	if err := batch.Execute(ctx, transaction, func(batchResults pgx.BatchResults) error {
		for inputIndex := range inputs {
			var event DayEvent
			if err := batchResults.QueryRow().Scan(
				&event.ID,
				&event.DayRecordID,
				&event.DeviceID,
				&event.ClientEventID,
				&event.EventType,
				&event.CategoryID,
				&event.Text,
				&event.OccurredAt,
				&event.OccurredAtLocal,
				&event.TargetClientEventID,
				&event.CorrectedAt,
				&event.CorrectedAtLocal,
			); errors.Is(err, pgx.ErrNoRows) {
				duplicateEventIDs = append(duplicateEventIDs, inputs[inputIndex].ClientEventID)
				continue
			} else if err != nil {
				return err
			}
			createdEvents = append(createdEvents, event)
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}
	return createdEvents, duplicateEventIDs, nil
}

func nullableEventReference(reference string) *string {
	if reference == "" {
		return nil
	}
	return &reference
}

func lockOffsetIfUnset(ctx context.Context, transaction pgx.Tx, dayRecordID int, locked bool, inputs []DayEventInput) error {
	if locked {
		return nil
	}
	for _, event := range inputs {
		if event.EventType != "transition" && event.EventType != "confirmation" && event.EventType != "note" {
			continue
		}
		offsetMinutes, err := extractOffsetMinutes(event.OccurredAtLocal)
		if err != nil {
			return ErrInvalidLocalTimestamp
		}
		_, err = transaction.Exec(ctx, `
			UPDATE day_records
			SET timezone_offset_minutes = $1, timezone_offset_locked = TRUE
			WHERE id = $2 AND timezone_offset_locked = FALSE
		`, offsetMinutes, dayRecordID)
		return err
	}
	return nil
}

func findOrCreateDayRecord(ctx context.Context, transaction pgx.Tx, userID int, calendarDate CalendarDate, existingDayRecordID int) (int, error) {
	if existingDayRecordID > 0 {
		return existingDayRecordID, nil
	}
	templateID, err := resolveTemplateForDateTx(ctx, transaction, userID, calendarDate)
	if err != nil {
		return 0, err
	}
	var snapshotID *int
	if templateID != nil {
		var latestSnapshotID int
		if snapshotError := transaction.QueryRow(ctx, `SELECT id FROM template_snapshots WHERE day_template_id = $1 ORDER BY snapshotted_at DESC, id DESC LIMIT 1`, *templateID).Scan(&latestSnapshotID); snapshotError == nil {
			snapshotID = &latestSnapshotID
		}
	}
	var dayRecordID int
	insertError := transaction.QueryRow(ctx, `
		INSERT INTO day_records(user_id, day_template_id, snapshot_id, calendar_date)
		VALUES($1, $2, $3, $4)
		ON CONFLICT (user_id, calendar_date) DO NOTHING
		RETURNING id
	`, userID, templateID, snapshotID, calendarDate).Scan(&dayRecordID)
	if errors.Is(insertError, pgx.ErrNoRows) {
		insertError = transaction.QueryRow(ctx, `
			SELECT id FROM day_records WHERE user_id = $1 AND calendar_date = $2
		`, userID, calendarDate).Scan(&dayRecordID)
	}
	if insertError != nil {
		return 0, insertError
	}
	return dayRecordID, nil
}

func (r *DayRecordRepository) ReplaceActualBlocks(ctx context.Context, dayRecordID, userID int, inputs []ActualBlockInput, clientOffsetMinutes *int) ([]ActualBlock, error) {
	transaction, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	err = transaction.QueryRow(ctx, `
		SELECT id FROM day_records WHERE id = $1 AND user_id = $2
	`, dayRecordID, userID).Scan(&dayRecordID)
	if err != nil {
		return nil, err
	}
	if clientOffsetMinutes != nil {
		if _, err := transaction.Exec(ctx, `UPDATE day_records SET timezone_offset_minutes = $1, timezone_offset_locked = TRUE WHERE id = $2 AND user_id = $3 AND timezone_offset_locked = FALSE`, *clientOffsetMinutes, dayRecordID, userID); err != nil {
			return nil, err
		}
	}

	_, err = transaction.Exec(ctx, `DELETE FROM actual_blocks WHERE day_record_id = $1`, dayRecordID)
	if err != nil {
		return nil, err
	}

	blocks := make([]ActualBlock, 0, len(inputs))
	now := time.Now()
	for _, input := range inputs {
		var block ActualBlock
		err := transaction.QueryRow(ctx, `
			INSERT INTO actual_blocks (day_record_id, category_id, block_type, start_time, duration_minutes, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, day_record_id, category_id, block_type, start_time, duration_minutes, updated_at
		`, dayRecordID, input.CategoryID, input.BlockType, input.StartTime, input.DurationMinutes, now).Scan(
			&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType, &block.StartTime, &block.DurationMinutes, &block.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	_, err = transaction.Exec(ctx, `UPDATE day_records SET updated_at = NOW() WHERE id = $1 AND user_id = $2`, dayRecordID, userID)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}

	return r.addUntrackedGaps(blocks), nil
}

func (r *DayRecordRepository) addUntrackedGaps(blocks []ActualBlock) []ActualBlock {
	if len(blocks) < 2 {
		return blocks
	}
	result := make([]ActualBlock, 0, len(blocks)*2)
	for index, block := range blocks {
		result = append(result, block)
		if index == len(blocks)-1 {
			continue
		}
		currentStart := block.StartTime
		nextStart := blocks[index+1].StartTime
		gapStart := currentStart.Add(time.Duration(block.DurationMinutes) * time.Minute)
		gapMinutes := int(nextStart.Sub(gapStart).Minutes())
		if gapMinutes > 0 {
			result = append(result, ActualBlock{
				DayRecordID:     block.DayRecordID,
				BlockType:       "untracked",
				StartTime:       gapStart,
				DurationMinutes: gapMinutes,
				UpdatedAt:       block.UpdatedAt,
			})
		}
	}
	return result
}

func (r *DayRecordRepository) recomputeActualBlocks(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
) ([]ActualBlock, []DayEvent, error) {
	batch := newPgBatchBuilder()
	batch.Queue(`
		SELECT id, day_record_id, device_id, client_event_id, event_type, category_id, text,
			occurred_at, COALESCE(occurred_at_local, ''), target_client_event_id,
			corrected_at, corrected_at_local
		FROM day_events
		WHERE day_record_id = $1
		ORDER BY id ASC
	`, dayRecordID)
	batch.Queue(`SELECT calendar_date FROM day_records WHERE id = $1`, dayRecordID)
	batch.Queue(`
		SELECT id, day_record_id, category_id, block_type, start_time, duration_minutes, updated_at
		FROM actual_blocks
		WHERE day_record_id = $1
		ORDER BY start_time ASC
	`, dayRecordID)

	var events []DayEvent
	var calendarDate CalendarDate
	var existingBlocks []ActualBlock
	if err := batch.Execute(ctx, transaction, func(batchResults pgx.BatchResults) error {
		var scanError error
		events, scanError = scanDayEvents(batchResults)
		if scanError != nil {
			return scanError
		}
		if scanError = batchResults.QueryRow().Scan(&calendarDate); scanError != nil {
			return scanError
		}
		existingBlocks, scanError = scanActualBlocks(batchResults)
		return scanError
	}); err != nil {
		return nil, nil, err
	}

	computedBlocks, err := computeIncrementalTimeline(events, existingBlocks, calendarDate)
	if err != nil {
		return nil, nil, err
	}

	blocks := make([]ActualBlock, 0, len(existingBlocks)+len(computedBlocks))
	blocks = append(blocks, existingBlocks...)
	insertedBlocks, err := insertComputedActualBlocks(ctx, transaction, dayRecordID, computedBlocks)
	if err != nil {
		return nil, nil, err
	}
	blocks = append(blocks, insertedBlocks...)

	return blocks, events, nil
}

func insertComputedActualBlocks(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
	computedBlocks []ComputedBlock,
) ([]ActualBlock, error) {
	insertedBlocks := make([]ActualBlock, 0, len(computedBlocks))
	if len(computedBlocks) == 0 {
		return insertedBlocks, nil
	}

	batch := newPgBatchBuilder()
	updatedAt := time.Now()
	for _, computed := range computedBlocks {
		batch.Queue(`
			INSERT INTO actual_blocks
				(day_record_id, category_id, block_type, start_time, duration_minutes, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, day_record_id, category_id, block_type, start_time,
				duration_minutes, updated_at
		`,
			dayRecordID,
			computed.CategoryID,
			computed.BlockType,
			computed.StartTime,
			computed.DurationMinutes,
			updatedAt,
		)
	}

	if err := batch.Execute(ctx, transaction, func(batchResults pgx.BatchResults) error {
		for range computedBlocks {
			var block ActualBlock
			if err := batchResults.QueryRow().Scan(
				&block.ID,
				&block.DayRecordID,
				&block.CategoryID,
				&block.BlockType,
				&block.StartTime,
				&block.DurationMinutes,
				&block.UpdatedAt,
			); err != nil {
				return err
			}
			insertedBlocks = append(insertedBlocks, block)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return insertedBlocks, nil
}
