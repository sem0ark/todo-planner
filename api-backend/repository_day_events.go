package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDeviceNotFound = NewNotFoundError("device not found")

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

func nullableEventReference(reference string) *string {
	if reference == "" {
		return nil
	}
	return &reference
}

type DayEventRepository struct {
	db *pgxpool.Pool
}

func NewDayEventRepository(db *pgxpool.Pool) *DayEventRepository {
	return &DayEventRepository{db: db}
}

func (repository *DayEventRepository) FindDeviceUserIDTx(
	ctx context.Context,
	transaction pgx.Tx,
	deviceID int,
) (int, error) {
	var userID int
	err := transaction.QueryRow(ctx, `SELECT user_id FROM devices WHERE id = $1`, deviceID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDeviceNotFound
	}
	return userID, err
}

func (repository *DayEventRepository) FindByRecordIDTx(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
) ([]DayEvent, error) {
	rows, err := transaction.Query(ctx, `
		SELECT id, day_record_id, device_id, client_event_id, event_type, category_id, text,
			occurred_at, COALESCE(occurred_at_local, ''), target_client_event_id,
			corrected_at, corrected_at_local
		FROM day_events
		WHERE day_record_id = $1
		ORDER BY id ASC
	`, dayRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]DayEvent, 0)
	for rows.Next() {
		var event DayEvent
		if err := rows.Scan(
			&event.ID, &event.DayRecordID, &event.DeviceID, &event.ClientEventID,
			&event.EventType, &event.CategoryID, &event.Text, &event.OccurredAt,
			&event.OccurredAtLocal, &event.TargetClientEventID, &event.CorrectedAt,
			&event.CorrectedAtLocal,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (repository *DayEventRepository) FindByRecordID(ctx context.Context, dayRecordID int) ([]DayEvent, error) {
	rows, err := repository.db.Query(ctx, `
		SELECT id, day_record_id, device_id, client_event_id, event_type, category_id, text,
			occurred_at, COALESCE(occurred_at_local, ''), target_client_event_id,
			corrected_at, corrected_at_local
		FROM day_events
		WHERE day_record_id = $1
		ORDER BY id ASC
	`, dayRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]DayEvent, 0)
	for rows.Next() {
		var event DayEvent
		if err := rows.Scan(
			&event.ID, &event.DayRecordID, &event.DeviceID, &event.ClientEventID,
			&event.EventType, &event.CategoryID, &event.Text, &event.OccurredAt,
			&event.OccurredAtLocal, &event.TargetClientEventID, &event.CorrectedAt,
			&event.CorrectedAtLocal,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (repository *DayEventRepository) FindExistingLocalTimestampsTx(
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
	return existingTimestamps, rows.Err()
}

func (repository *DayEventRepository) InsertTx(
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
			err := batchResults.QueryRow().Scan(
				&event.ID, &event.DayRecordID, &event.DeviceID, &event.ClientEventID,
				&event.EventType, &event.CategoryID, &event.Text, &event.OccurredAt,
				&event.OccurredAtLocal, &event.TargetClientEventID, &event.CorrectedAt,
				&event.CorrectedAtLocal,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				duplicateEventIDs = append(duplicateEventIDs, inputs[inputIndex].ClientEventID)
				continue
			}
			if err != nil {
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

func (repository *DayEventRepository) InsertManualTx(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
	inputs []DayEventInput,
) ([]DayEvent, error) {
	createdEvents := make([]DayEvent, 0, len(inputs))
	for _, input := range inputs {
		var event DayEvent
		if err := transaction.QueryRow(ctx, `
			INSERT INTO day_events (day_record_id, event_type, category_id, text, occurred_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, day_record_id, event_type, category_id, text, occurred_at
		`, dayRecordID, input.EventType, input.CategoryID, input.Text, input.OccurredAt).Scan(
			&event.ID, &event.DayRecordID, &event.EventType, &event.CategoryID,
			&event.Text, &event.OccurredAt,
		); err != nil {
			return nil, err
		}
		createdEvents = append(createdEvents, event)
	}
	return createdEvents, nil
}
