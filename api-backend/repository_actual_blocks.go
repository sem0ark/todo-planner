package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActualBlock struct {
	ID              int          `json:"id"`
	DayRecordID     int          `json:"day_record_id"`
	CategoryID      *int         `json:"category_id"`
	BlockType       string       `json:"block_type"` // actual | blank | untracked
	StartTime       ScheduleTime `json:"-"`
	DurationMinutes int          `json:"duration_minutes"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type ActualBlockRepository struct {
	db *pgxpool.Pool
}

func NewActualBlockRepository(db *pgxpool.Pool) *ActualBlockRepository {
	return &ActualBlockRepository{db: db}
}

func (repository *ActualBlockRepository) FindByRecordIDTx(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
) ([]ActualBlock, error) {
	rows, err := transaction.Query(ctx, `
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
		if err := rows.Scan(
			&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType,
			&block.StartTime, &block.DurationMinutes, &block.UpdatedAt,
		); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

func (repository *ActualBlockRepository) FindByRecordID(ctx context.Context, dayRecordID int) ([]ActualBlock, error) {
	rows, err := repository.db.Query(ctx, `
		SELECT id, day_record_id, category_id, block_type, start_time,
			duration_minutes, updated_at
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
		if err := rows.Scan(
			&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType,
			&block.StartTime, &block.DurationMinutes, &block.UpdatedAt,
		); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

func (repository *ActualBlockRepository) InsertComputedTx(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
	computedBlocks []ComputedBlock,
) ([]ActualBlock, error) {
	insertedBlocks := make([]ActualBlock, 0, len(computedBlocks))
	if len(computedBlocks) == 0 {
		return insertedBlocks, nil
	}

	updatedAt := time.Now()
	batch := newPgBatchBuilder()
	for _, computedBlock := range computedBlocks {
		batch.Queue(`
			INSERT INTO actual_blocks
				(day_record_id, category_id, block_type, start_time, duration_minutes, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, day_record_id, category_id, block_type, start_time,
				duration_minutes, updated_at
		`, dayRecordID, computedBlock.CategoryID, computedBlock.BlockType,
			computedBlock.StartTime, computedBlock.DurationMinutes, updatedAt)
	}

	if err := batch.Execute(ctx, transaction, func(batchResults pgx.BatchResults) error {
		for range computedBlocks {
			var block ActualBlock
			if err := batchResults.QueryRow().Scan(
				&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType,
				&block.StartTime, &block.DurationMinutes, &block.UpdatedAt,
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

func (repository *ActualBlockRepository) UpdateDurationTx(
	ctx context.Context,
	transaction pgx.Tx,
	blockID int,
	durationMinutes int,
) error {
	_, err := transaction.Exec(ctx, `
		UPDATE actual_blocks
		SET duration_minutes = $2, updated_at = $3
		WHERE id = $1
	`, blockID, durationMinutes, time.Now())
	return err
}

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
	updatedAt := time.Now()
	for _, input := range inputs {
		var block ActualBlock
		if err := transaction.QueryRow(ctx, `
			INSERT INTO actual_blocks (day_record_id, category_id, block_type, start_time, duration_minutes, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, day_record_id, category_id, block_type, start_time, duration_minutes, updated_at
		`, dayRecordID, input.CategoryID, input.BlockType, input.StartTime, input.DurationMinutes, updatedAt).Scan(
			&block.ID, &block.DayRecordID, &block.CategoryID, &block.BlockType,
			&block.StartTime, &block.DurationMinutes, &block.UpdatedAt,
		); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}
