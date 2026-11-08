package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DayRecordRepository struct {
	db *pgxpool.Pool
}

type DayRecord struct {
	ID                    int             `json:"id"`
	UserID                int             `json:"user_id"`
	SnapshotID            *int            `json:"snapshot_id"`
	DayTemplateID         *int            `json:"day_template_id"`
	CalendarDate          CalendarDate    `json:"calendar_date"`
	TimezoneOffsetMinutes int             `json:"timezone_offset_minutes"`
	BlocksEdited          bool            `json:"blocks_edited"`
	SnapshotBlocks        []SnapshotBlock `json:"snapshot_blocks"`
	ActualBlocks          []ActualBlock   `json:"actual_blocks"`
	Events                []DayEvent      `json:"events"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	SnapshottedAt         time.Time       `json:"-"`
}

var (
	ErrDayRecordPast          = NewBadRequestError("day record is in the past")
	ErrDayRecordNotFound      = NewAppErrorWithCause(http.StatusNotFound, "day record not found", pgx.ErrNoRows)
	ErrDayRecordAlreadyExists = NewConflictError("day record already exists")
)

const dayRecordColumns = `id, user_id, day_template_id, snapshot_id, calendar_date,
	timezone_offset_minutes, blocks_edited, created_at, updated_at`

func scanDayRecord(row pgx.Row, record *DayRecord) error {
	return row.Scan(
		&record.ID, &record.UserID, &record.DayTemplateID, &record.SnapshotID,
		&record.CalendarDate, &record.TimezoneOffsetMinutes, &record.BlocksEdited,
		&record.CreatedAt, &record.UpdatedAt,
	)
}

func NewDayRecordRepository(db *pgxpool.Pool) *DayRecordRepository {
	return &DayRecordRepository{db: db}
}

func (repository *DayRecordRepository) EnsureOwned(ctx context.Context, transaction dbQuerier, recordID, userID int) error {
	err := transaction.QueryRow(ctx, `
		SELECT id FROM day_records WHERE id = $1 AND user_id = $2
	`, recordID, userID).Scan(&recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDayRecordNotFound
	}
	return err
}

func (repository *DayRecordRepository) MarkBlocksEdited(ctx context.Context, transaction dbQuerier, recordID, userID int) error {
	_, err := transaction.Exec(ctx, `
		UPDATE day_records SET blocks_edited = TRUE, updated_at = NOW() WHERE id = $1 AND user_id = $2
	`, recordID, userID)
	return err
}

// UpdateTimezoneOffset only changes how clients present the day's UTC instants.
func (repository *DayRecordRepository) UpdateTimezoneOffset(
	ctx context.Context,
	userID int,
	calendarDate CalendarDate,
	offsetMinutes int,
) (int, error) {
	var recordID int
	err := repository.db.QueryRow(ctx, `
		UPDATE day_records SET timezone_offset_minutes = $1, updated_at = NOW()
		WHERE user_id = $2 AND calendar_date = $3
		RETURNING id
	`, offsetMinutes, userID, calendarDate).Scan(&recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDayRecordNotFound
	}
	return recordID, err
}

// FindIDForDate returns ErrDayRecordNotFound when the day was never created.
func (repository *DayRecordRepository) FindIDForDate(
	ctx context.Context,
	transaction dbQuerier,
	userID int,
	calendarDate CalendarDate,
) (int, error) {
	var recordID int
	err := transaction.QueryRow(ctx, `
		SELECT id FROM day_records WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDayRecordNotFound
	}
	return recordID, err
}

func (repository *DayRecordRepository) FindOrCreate(
	ctx context.Context,
	transaction dbQuerier,
	userID int,
	calendarDate CalendarDate,
	offsetMinutes int,
	templateID *int,
	snapshotID *int,
) (int, error) {
	var recordID int
	err := transaction.QueryRow(ctx, `
		INSERT INTO day_records(user_id, day_template_id, snapshot_id, calendar_date, timezone_offset_minutes)
		VALUES($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, calendar_date) DO NOTHING
		RETURNING id
	`, userID, templateID, snapshotID, calendarDate, offsetMinutes).Scan(&recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = transaction.QueryRow(ctx, `
			SELECT id FROM day_records WHERE user_id = $1 AND calendar_date = $2
		`, userID, calendarDate).Scan(&recordID)
	}
	return recordID, err
}

func (repository *DayRecordRepository) FindByDateRange(ctx context.Context, userID int, fromDate, toDate CalendarDate) ([]DayRecord, error) {
	rows, err := repository.db.Query(ctx, `
		SELECT `+dayRecordColumns+`
		FROM day_records
		WHERE user_id = $1 AND calendar_date >= $2 AND calendar_date <= $3
		ORDER BY calendar_date ASC
	`, userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []DayRecord
	for rows.Next() {
		var record DayRecord
		if err := scanDayRecord(rows, &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (repository *DayRecordRepository) FindByDate(ctx context.Context, userID int, calendarDate CalendarDate) (*DayRecord, error) {
	var record DayRecord
	err := scanDayRecord(repository.db.QueryRow(ctx, `
		SELECT `+dayRecordColumns+`
		FROM day_records
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate), &record)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDayRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// FindByID returns a day record only when it belongs to the requested user.
func (repository *DayRecordRepository) FindByID(ctx context.Context, id, userID int) (*DayRecord, error) {
	var record DayRecord
	err := scanDayRecord(repository.db.QueryRow(ctx, `
		SELECT `+dayRecordColumns+`
		FROM day_records
		WHERE id = $1 AND user_id = $2
	`, id, userID), &record)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDayRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (repository *DayRecordRepository) Create(
	ctx context.Context,
	transaction dbQuerier,
	userID int,
	calendarDate CalendarDate,
	offsetMinutes int,
	templateID *int,
	snapshotID *int,
) (*DayRecord, error) {
	var record DayRecord
	err := scanDayRecord(transaction.QueryRow(ctx, `
		INSERT INTO day_records (user_id, day_template_id, snapshot_id, calendar_date, timezone_offset_minutes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+dayRecordColumns, userID, templateID, snapshotID, calendarDate, offsetMinutes), &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (repository *DayRecordRepository) UpdateTemplate(
	ctx context.Context,
	transaction dbQuerier,
	userID int,
	calendarDate CalendarDate,
	templateID *int,
	snapshotID *int,
) (int, error) {
	var recordID int
	err := transaction.QueryRow(ctx, `
		UPDATE day_records
		SET day_template_id = $1, snapshot_id = $2, updated_at = NOW()
		WHERE user_id = $3 AND calendar_date = $4 AND calendar_date >= CURRENT_DATE
		RETURNING id
	`, templateID, snapshotID, userID, calendarDate).Scan(&recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if checkError := transaction.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM day_records WHERE user_id = $1 AND calendar_date = $2)
		`, userID, calendarDate).Scan(&exists); checkError != nil {
			return 0, checkError
		} else if !exists {
			return 0, ErrDayRecordNotFound
		}
		return 0, ErrDayRecordPast
	}
	if err != nil {
		return 0, err
	}
	return recordID, nil
}

func (repository *DayRecordRepository) TouchAndFindByID(
	ctx context.Context,
	transaction dbQuerier,
	dayRecordID int,
	userID int,
) (*DayRecord, error) {
	record := &DayRecord{}
	err := scanDayRecord(transaction.QueryRow(ctx, `
		UPDATE day_records SET updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+dayRecordColumns, dayRecordID, userID), record)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDayRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}
