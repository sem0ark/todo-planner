package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChangeLogRepository struct {
	db *pgxpool.Pool
}

type ChangeLogEntry struct {
	EntityType string    `json:"entity_type"` // category | template_group | day_template | weekly_schedule | schedule_override | day_record | settings
	EntityID   int       `json:"entity_id"`
	Operation  string    `json:"operation"` // create | update | delete
	OccurredAt time.Time `json:"occurred_at"`
}

func NewChangeLogRepository(db *pgxpool.Pool) *ChangeLogRepository {
	return &ChangeLogRepository{db: db}
}

func (r *ChangeLogRepository) RecordChanges(ctx context.Context, deviceID int, userID int, changes []ChangeLogEntry) error {
	if len(changes) == 0 {
		return nil
	}

	batch := make([][]interface{}, 0, len(changes))
	for _, change := range changes {
		batch = append(batch, []interface{}{
			deviceID,
			userID,
			change.EntityType,
			change.EntityID,
			change.Operation,
			change.OccurredAt,
		})
	}

	_, err := r.db.CopyFrom(
		ctx,
		[]string{"change_log"},
		[]string{"device_id", "user_id", "entity_type", "entity_id", "operation", "occurred_at"},
		&batchRows{rows: batch},
	)

	return err
}

func (r *ChangeLogRepository) FetchChangesSince(ctx context.Context, userID int, excludeDeviceID int, since *time.Time) ([]ChangeLogEntry, error) {
	var rows []ChangeLogEntry

	query := `
		SELECT entity_type, entity_id, operation, occurred_at
		FROM change_log
		WHERE user_id = $1 AND device_id != $2
	`

	args := []interface{}{userID, excludeDeviceID}

	if since != nil {
		query += ` AND occurred_at > $3`
		args = append(args, since)
	}

	query += ` ORDER BY occurred_at ASC`

	dbRows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer dbRows.Close()

	for dbRows.Next() {
		var entry ChangeLogEntry
		if err := dbRows.Scan(
			&entry.EntityType,
			&entry.EntityID,
			&entry.Operation,
			&entry.OccurredAt,
		); err != nil {
			return nil, err
		}
		rows = append(rows, entry)
	}

	if err := dbRows.Err(); err != nil {
		return nil, err
	}

	return rows, nil
}

type batchRows struct {
	rows [][]interface{}
	idx  int
}

func (b *batchRows) Next() bool {
	b.idx++
	return b.idx <= len(b.rows)
}

func (b *batchRows) Values() ([]interface{}, error) {
	return b.rows[b.idx-1], nil
}

func (b *batchRows) Err() error {
	return nil
}
