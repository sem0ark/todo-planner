package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDayTemplateNotFound = errors.New("day template not found")

// A template for a day's schedule
type DayTemplate struct {
	ID              int            `json:"id"`
	UserID          int            `json:"user_id"`
	TemplateGroupID *int           `json:"template_group_id"`
	Name            string         `json:"name"`
	PlannedBlocks   []PlannedBlock `json:"planned_blocks"`
	IsDeleted       bool           `json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// A time block in a day template
type PlannedBlock struct {
	ID              int    `json:"id"`
	DayTemplateID   int    `json:"day_template_id"`
	CategoryID      int    `json:"category_id"`
	StartTime       string `json:"start_time"` // HH:MM:SS
	DurationMinutes int    `json:"duration_minutes"`
}

// Day template creation/update request data
type DayTemplateInput struct {
	Name            string              `json:"name"`
	TemplateGroupID *int                `json:"template_group_id"`
	PlannedBlocks   []PlannedBlockInput `json:"planned_blocks"`
}

// Planned block input data
type PlannedBlockInput struct {
	CategoryID      int    `json:"category_id"`
	StartTime       string `json:"start_time"` // HH:MM:SS
	DurationMinutes int    `json:"duration_minutes"`
}

type DayTemplateRepository struct {
	db *pgxpool.Pool
}

func NewDayTemplateRepository(db *pgxpool.Pool) *DayTemplateRepository {
	return &DayTemplateRepository{db: db}
}

func (r *DayTemplateRepository) FindByUser(ctx context.Context, userID int) ([]DayTemplate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, template_group_id, name, created_at, updated_at
		FROM day_templates
		WHERE user_id = $1 AND is_deleted = FALSE
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]DayTemplate, 0)
	for rows.Next() {
		var template DayTemplate
		if err := rows.Scan(&template.ID, &template.UserID, &template.TemplateGroupID, &template.Name, &template.CreatedAt, &template.UpdatedAt); err != nil {
			return nil, err
		}

		blocks, err := r.findPlannedBlocks(ctx, template.ID)
		if err != nil {
			return nil, err
		}
		template.PlannedBlocks = blocks

		templates = append(templates, template)
	}

	return templates, rows.Err()
}

func (r *DayTemplateRepository) FindByID(ctx context.Context, id, userID int) (*DayTemplate, error) {
	var template DayTemplate
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, template_group_id, name, is_deleted, created_at, updated_at
		FROM day_templates
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&template.ID, &template.UserID, &template.TemplateGroupID, &template.Name, &template.IsDeleted, &template.CreatedAt, &template.UpdatedAt)

	if err != nil {
		return nil, err
	}

	blocks, err := r.findPlannedBlocks(ctx, template.ID)
	if err != nil {
		return nil, err
	}
	template.PlannedBlocks = blocks

	return &template, nil
}

func (r *DayTemplateRepository) findPlannedBlocks(ctx context.Context, templateID int) ([]PlannedBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, day_template_id, category_id, start_time, duration_minutes
		FROM planned_blocks
		WHERE day_template_id = $1
		ORDER BY start_time ASC
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]PlannedBlock, 0)
	for rows.Next() {
		var block PlannedBlock
		if err := rows.Scan(&block.ID, &block.DayTemplateID, &block.CategoryID, &block.StartTime, &block.DurationMinutes); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, rows.Err()
}

func (r *DayTemplateRepository) Create(ctx context.Context, input DayTemplateInput, userID int) (*DayTemplate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var template DayTemplate
	err = tx.QueryRow(ctx, `
		INSERT INTO day_templates (user_id, template_group_id, name, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, FALSE, NOW(), NOW())
		RETURNING id, user_id, template_group_id, name, created_at, updated_at
	`, userID, input.TemplateGroupID, input.Name).Scan(&template.ID, &template.UserID, &template.TemplateGroupID, &template.Name, &template.CreatedAt, &template.UpdatedAt)

	if err != nil {
		return nil, err
	}

	blocks := make([]PlannedBlock, 0, len(input.PlannedBlocks))
	for _, blockInput := range input.PlannedBlocks {
		var block PlannedBlock
		err = tx.QueryRow(ctx, `
			INSERT INTO planned_blocks (day_template_id, category_id, start_time, duration_minutes)
			VALUES ($1, $2, $3, $4)
			RETURNING id, day_template_id, category_id, start_time, duration_minutes
		`, template.ID, blockInput.CategoryID, blockInput.StartTime, blockInput.DurationMinutes).Scan(
			&block.ID, &block.DayTemplateID, &block.CategoryID, &block.StartTime, &block.DurationMinutes,
		)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	template.PlannedBlocks = blocks

	// Create snapshot of the template
	if err := r.createSnapshot(ctx, tx, template.ID, userID, blocks); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &template, nil
}

func (r *DayTemplateRepository) Update(ctx context.Context, id int, input DayTemplateInput, userID int) (*DayTemplate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var template DayTemplate
	err = tx.QueryRow(ctx, `
		UPDATE day_templates
		SET name = $1, template_group_id = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4 AND is_deleted = FALSE
		RETURNING id, user_id, template_group_id, name, created_at, updated_at
	`, input.Name, input.TemplateGroupID, id, userID).Scan(&template.ID, &template.UserID, &template.TemplateGroupID, &template.Name, &template.CreatedAt, &template.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, ErrDayTemplateNotFound
	}
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `DELETE FROM planned_blocks WHERE day_template_id = $1`, id)
	if err != nil {
		return nil, err
	}

	blocks := make([]PlannedBlock, 0, len(input.PlannedBlocks))
	for _, blockInput := range input.PlannedBlocks {
		var block PlannedBlock
		err = tx.QueryRow(ctx, `
			INSERT INTO planned_blocks (day_template_id, category_id, start_time, duration_minutes)
			VALUES ($1, $2, $3, $4)
			RETURNING id, day_template_id, category_id, start_time, duration_minutes
		`, template.ID, blockInput.CategoryID, blockInput.StartTime, blockInput.DurationMinutes).Scan(
			&block.ID, &block.DayTemplateID, &block.CategoryID, &block.StartTime, &block.DurationMinutes,
		)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	template.PlannedBlocks = blocks

	// Create new snapshot of the updated template
	if err := r.createSnapshot(ctx, tx, template.ID, userID, blocks); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &template, nil
}

// createSnapshot creates a snapshot of a template's planned blocks
func (r *DayTemplateRepository) createSnapshot(ctx context.Context, tx pgx.Tx, templateID, userID int, blocks []PlannedBlock) error {
	// Create snapshot entry
	var snapshotID int
	err := tx.QueryRow(ctx, `
		INSERT INTO template_snapshots (day_template_id, user_id, snapshotted_at)
		VALUES ($1, $2, NOW())
		RETURNING id
	`, templateID, userID).Scan(&snapshotID)
	if err != nil {
		return err
	}

	// Copy all planned blocks to snapshot_blocks
	for _, block := range blocks {
		_, err = tx.Exec(ctx, `
			INSERT INTO snapshot_blocks (snapshot_id, category_id, start_time, duration_minutes)
			VALUES ($1, $2, $3, $4)
		`, snapshotID, block.CategoryID, block.StartTime, block.DurationMinutes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *DayTemplateRepository) Delete(ctx context.Context, id, userID int) error {
	result, err := r.db.Exec(ctx, `
		UPDATE day_templates
		SET is_deleted = TRUE, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`, id, userID)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrDayTemplateNotFound
	}

	return nil
}
