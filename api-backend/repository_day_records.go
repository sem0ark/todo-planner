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
	TimezoneOffsetMinutes *int            `json:"timezone_offset_minutes"`
	TimezoneOffsetLocked  bool            `json:"timezone_offset_locked"`
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

func NewDayRecordRepository(db *pgxpool.Pool) *DayRecordRepository {
	return &DayRecordRepository{db: db}
}

func (repository *DayRecordRepository) EnsureOwnedTx(ctx context.Context, transaction pgx.Tx, recordID, userID int) error {
	err := transaction.QueryRow(ctx, `
		SELECT id FROM day_records WHERE id = $1 AND user_id = $2
	`, recordID, userID).Scan(&recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDayRecordNotFound
	}
	return err
}

func (repository *DayRecordRepository) FindCalendarDateTx(ctx context.Context, transaction pgx.Tx, recordID int) (CalendarDate, error) {
	var calendarDate CalendarDate
	err := transaction.QueryRow(ctx, `SELECT calendar_date FROM day_records WHERE id = $1`, recordID).Scan(&calendarDate)
	return calendarDate, err
}

func (repository *DayRecordRepository) TouchTx(ctx context.Context, transaction pgx.Tx, recordID, userID int) error {
	_, err := transaction.Exec(ctx, `
		UPDATE day_records SET updated_at = NOW() WHERE id = $1 AND user_id = $2
	`, recordID, userID)
	return err
}

func (repository *DayRecordRepository) LockTimezoneOffsetTx(
	ctx context.Context,
	transaction pgx.Tx,
	recordID, userID, offsetMinutes int,
) error {
	_, err := transaction.Exec(ctx, `
		UPDATE day_records
		SET timezone_offset_minutes = $1, timezone_offset_locked = TRUE
		WHERE id = $2 AND user_id = $3 AND timezone_offset_locked = FALSE
	`, offsetMinutes, recordID, userID)
	return err
}

func (repository *DayRecordRepository) FindForDateTx(
	ctx context.Context,
	transaction pgx.Tx,
	userID int,
	calendarDate CalendarDate,
) (int, bool, error) {
	var recordID int
	var timezoneOffsetLocked bool
	err := transaction.QueryRow(ctx, `
		SELECT id, timezone_offset_locked
		FROM day_records
		WHERE user_id = $1 AND calendar_date = $2
	`, userID, calendarDate).Scan(&recordID, &timezoneOffsetLocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return recordID, timezoneOffsetLocked, err
}

func (repository *DayRecordRepository) FindOrCreateTx(
	ctx context.Context,
	transaction pgx.Tx,
	userID int,
	calendarDate CalendarDate,
	templateID *int,
	snapshotID *int,
) (int, error) {
	var recordID int
	err := transaction.QueryRow(ctx, `
		INSERT INTO day_records(user_id, day_template_id, snapshot_id, calendar_date)
		VALUES($1, $2, $3, $4)
		ON CONFLICT (user_id, calendar_date) DO NOTHING
		RETURNING id
	`, userID, templateID, snapshotID, calendarDate).Scan(&recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = transaction.QueryRow(ctx, `
			SELECT id FROM day_records WHERE user_id = $1 AND calendar_date = $2
		`, userID, calendarDate).Scan(&recordID)
	}
	return recordID, err
}

func (repository *DayRecordRepository) FindByDateRange(ctx context.Context, userID int, fromDate, toDate CalendarDate) ([]DayRecord, error) {
	rows, err := repository.db.Query(ctx, `
		SELECT id, user_id, day_template_id, snapshot_id, calendar_date,
			timezone_offset_minutes, timezone_offset_locked, created_at, updated_at
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
		if err := rows.Scan(
			&record.ID, &record.UserID, &record.DayTemplateID, &record.SnapshotID,
			&record.CalendarDate, &record.TimezoneOffsetMinutes,
			&record.TimezoneOffsetLocked, &record.CreatedAt, &record.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (repository *DayRecordRepository) FindByDate(ctx context.Context, userID int, calendarDate CalendarDate) (*DayRecord, error) {
	var record DayRecord
	err := repository.db.QueryRow(ctx, `
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
	return &record, nil
}

// FindByID returns a day record only when it belongs to the requested user.
func (repository *DayRecordRepository) FindByID(ctx context.Context, id, userID int) (*DayRecord, error) {
	var record DayRecord
	err := repository.db.QueryRow(ctx, `
		SELECT id, user_id, day_template_id, snapshot_id, calendar_date,
			timezone_offset_minutes, timezone_offset_locked, created_at, updated_at
		FROM day_records
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&record.ID, &record.UserID, &record.DayTemplateID, &record.SnapshotID, &record.CalendarDate, &record.TimezoneOffsetMinutes, &record.TimezoneOffsetLocked, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDayRecordNotFound
		}
		return nil, err
	}

	return &record, nil
}

func (repository *DayRecordRepository) CreateTx(
	ctx context.Context,
	transaction pgx.Tx,
	userID int,
	calendarDate CalendarDate,
	templateID *int,
	snapshotID *int,
) (*DayRecord, error) {
	var record DayRecord
	now := time.Now()
	err := transaction.QueryRow(ctx, `
		INSERT INTO day_records (user_id, day_template_id, snapshot_id, calendar_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, day_template_id, snapshot_id, calendar_date,
			timezone_offset_minutes, timezone_offset_locked, created_at, updated_at
	`, userID, templateID, snapshotID, calendarDate, now, now).Scan(
		&record.ID, &record.UserID, &record.DayTemplateID, &record.SnapshotID, &record.CalendarDate,
		&record.TimezoneOffsetMinutes, &record.TimezoneOffsetLocked, &record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (repository *DayRecordRepository) UpdateTemplateTx(
	ctx context.Context,
	transaction pgx.Tx,
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

func (repository *DayRecordRepository) TouchAndFindByIDTx(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
	userID int,
) (*DayRecord, error) {
	record := &DayRecord{}
	err := transaction.QueryRow(ctx, `
		UPDATE day_records SET updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, day_template_id, snapshot_id, calendar_date,
			timezone_offset_minutes, timezone_offset_locked, created_at, updated_at
	`, dayRecordID, userID).Scan(
		&record.ID, &record.UserID, &record.DayTemplateID, &record.SnapshotID,
		&record.CalendarDate, &record.TimezoneOffsetMinutes,
		&record.TimezoneOffsetLocked, &record.CreatedAt, &record.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDayRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}
