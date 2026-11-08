package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActualBlock is a stored span of tracked time. StartAt is a UTC instant;
// clients render it with the day's timezone_offset_minutes.
type ActualBlock struct {
	ID              int       `json:"id"`
	DayRecordID     int       `json:"day_record_id"`
	CategoryID      *int      `json:"category_id"`
	BlockType       string    `json:"block_type"` // always "actual"
	StartAt         time.Time `json:"start_at"`
	DurationMinutes int       `json:"duration_minutes"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ActualBlockInput is a block before it is stored, either derived from events
// or submitted during review.
type ActualBlockInput struct {
	CategoryID      *int      `json:"category_id"`
	BlockType       string    `json:"block_type"`
	StartAt         time.Time `json:"start_at"`
	DurationMinutes int       `json:"duration_minutes"`
}

func (block ActualBlockInput) endAt() time.Time {
	return block.StartAt.Add(time.Duration(block.DurationMinutes) * time.Minute)
}

type ActualBlockRepository struct {
	db *pgxpool.Pool
}

func NewActualBlockRepository(db *pgxpool.Pool) *ActualBlockRepository {
	return &ActualBlockRepository{db: db}
}

func (repository *ActualBlockRepository) FindByRecordID(ctx context.Context, db dbQuerier, dayRecordID int) ([]ActualBlock, error) {
	rows, err := db.Query(ctx, `
		SELECT id, day_record_id, category_id, block_type, start_at, duration_minutes, updated_at
		FROM actual_blocks
		WHERE day_record_id = $1
		ORDER BY start_at ASC, id ASC
	`, dayRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]ActualBlock, 0)
	for rows.Next() {
		var block ActualBlock
		if err := rows.Scan(
			&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType,
			&block.StartAt, &block.DurationMinutes, &block.UpdatedAt,
		); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

// ReplaceTx swaps the day's complete block list for inputs.
func (repository *ActualBlockRepository) ReplaceTx(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
	inputs []ActualBlockInput,
) ([]ActualBlock, error) {
	if _, err := transaction.Exec(ctx, `DELETE FROM actual_blocks WHERE day_record_id = $1`, dayRecordID); err != nil {
		return nil, err
	}
	blocks := make([]ActualBlock, 0, len(inputs))
	if len(inputs) == 0 {
		return blocks, nil
	}

	updatedAt := time.Now()
	batch := newPgBatchBuilder()
	for _, input := range inputs {
		batch.Queue(`
			INSERT INTO actual_blocks
				(day_record_id, category_id, block_type, start_at, duration_minutes, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, day_record_id, category_id, block_type, start_at, duration_minutes, updated_at
		`, dayRecordID, input.CategoryID, input.BlockType, input.StartAt.UTC(), input.DurationMinutes, updatedAt)
	}
	if err := batch.Execute(ctx, transaction, func(batchResults pgx.BatchResults) error {
		for range inputs {
			var block ActualBlock
			if err := batchResults.QueryRow().Scan(
				&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType,
				&block.StartAt, &block.DurationMinutes, &block.UpdatedAt,
			); err != nil {
				return err
			}
			blocks = append(blocks, block)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return blocks, nil
}
