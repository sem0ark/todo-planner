package main

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDayTemplateNotFound = errors.New("day template not found")
var ErrInvalidTemplateBlock = errors.New("invalid snapshot block")
var ErrTemplateCategoryNotFound = errors.New("unknown category_id")

var templateTimePattern = regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9](?::[0-5][0-9])?$`)

// DayTemplate contains metadata and the latest immutable snapshot.
type DayTemplate struct {
	ID              int               `json:"id"`
	UserID          int               `json:"-"`
	TemplateGroupID *int              `json:"template_group_id"`
	Name            string            `json:"name"`
	CurrentSnapshot *TemplateSnapshot `json:"current_snapshot"`
	IsDeleted       bool              `json:"-"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// DayTemplateInput is shared by template creation and snapshot updates.
type DayTemplateInput struct {
	Name            string               `json:"name"`
	TemplateGroupID *int                 `json:"template_group_id"`
	SnapshotBlocks  []SnapshotBlockInput `json:"snapshot_blocks"`
}

// SnapshotBlockInput describes a block in a newly-created snapshot.
type SnapshotBlockInput struct {
	CategoryID      int    `json:"category_id"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
}

type DayTemplateRepository struct {
	db *pgxpool.Pool
}

func NewDayTemplateRepository(db *pgxpool.Pool) *DayTemplateRepository {
	return &DayTemplateRepository{db: db}
}

func (r *DayTemplateRepository) validateInput(ctx context.Context, input DayTemplateInput, userID int) error {
	for _, block := range input.SnapshotBlocks {
		if !templateTimePattern.MatchString(block.StartTime) ||
			block.DurationMinutes < 30 || block.DurationMinutes%15 != 0 {
			return ErrInvalidTemplateBlock
		}

		var categoryExists bool
		err := r.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM block_categories
				WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
			)
		`, block.CategoryID, userID).Scan(&categoryExists)
		if err != nil {
			return err
		}
		if !categoryExists {
			return ErrTemplateCategoryNotFound
		}
	}

	if input.TemplateGroupID != nil {
		var groupExists bool
		err := r.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM template_groups
				WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
			)
		`, *input.TemplateGroupID, userID).Scan(&groupExists)
		if err != nil {
			return err
		}
		if !groupExists {
			return ErrTemplateGroupNotFound
		}
	}

	return nil
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
		if err := r.loadCurrentSnapshot(ctx, &template); err != nil {
			return nil, err
		}
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
	if err := r.loadCurrentSnapshot(ctx, &template); err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *DayTemplateRepository) loadCurrentSnapshot(ctx context.Context, template *DayTemplate) error {
	var snapshot TemplateSnapshot
	err := r.db.QueryRow(ctx, `
		SELECT id, day_template_id, user_id, snapshotted_at
		FROM template_snapshots
		WHERE day_template_id = $1 AND user_id = $2
		ORDER BY snapshotted_at DESC, id DESC
		LIMIT 1
	`, template.ID, template.UserID).Scan(&snapshot.ID, &snapshot.DayTemplateID, &snapshot.UserID, &snapshot.SnapshottedAt)
	if err == pgx.ErrNoRows {
		template.CurrentSnapshot = nil
		return nil
	}
	if err != nil {
		return err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, snapshot_id, category_id, start_time, duration_minutes
		FROM snapshot_blocks
		WHERE snapshot_id = $1
		ORDER BY start_time ASC, id ASC
	`, snapshot.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	snapshot.SnapshotBlocks = make([]SnapshotBlock, 0)
	for rows.Next() {
		var block SnapshotBlock
		if err := rows.Scan(&block.ID, &block.SnapshotID, &block.CategoryID, &block.StartTime, &block.DurationMinutes); err != nil {
			return err
		}
		snapshot.SnapshotBlocks = append(snapshot.SnapshotBlocks, block)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	template.CurrentSnapshot = &snapshot
	return nil
}

func (r *DayTemplateRepository) Create(ctx context.Context, input DayTemplateInput, userID int) (*DayTemplate, error) {
	if err := r.validateInput(ctx, input, userID); err != nil {
		return nil, err
	}

	transaction, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	var templateID int
	err = transaction.QueryRow(ctx, `
		INSERT INTO day_templates (user_id, template_group_id, name)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, input.TemplateGroupID, input.Name).Scan(&templateID)
	if err != nil {
		return nil, err
	}

	if err := createTemplateSnapshot(ctx, transaction, templateID, userID, input.SnapshotBlocks); err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, templateID, userID)
}

func (r *DayTemplateRepository) Update(ctx context.Context, id int, input DayTemplateInput, userID int) (*DayTemplate, error) {
	if err := r.validateInput(ctx, input, userID); err != nil {
		return nil, err
	}

	transaction, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	var templateID int
	err = transaction.QueryRow(ctx, `
		UPDATE day_templates
		SET name = $1, template_group_id = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4 AND is_deleted = FALSE
		RETURNING id
	`, input.Name, input.TemplateGroupID, id, userID).Scan(&templateID)
	if err == pgx.ErrNoRows {
		return nil, ErrDayTemplateNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := createTemplateSnapshot(ctx, transaction, templateID, userID, input.SnapshotBlocks); err != nil {
		return nil, err
	}
	_, err = transaction.Exec(ctx, `
		UPDATE day_records
		SET snapshot_id = (
			SELECT id FROM template_snapshots
			WHERE day_template_id = $1
			ORDER BY snapshotted_at DESC, id DESC
			LIMIT 1
		), updated_at = NOW()
		WHERE user_id = $2 AND day_template_id = $1 AND calendar_date >= CURRENT_DATE
	`, id, userID)
	if err != nil {
		return nil, err
	}

	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, templateID, userID)
}

func createTemplateSnapshot(ctx context.Context, transaction pgx.Tx, templateID, userID int, blocks []SnapshotBlockInput) error {
	var snapshotID int
	err := transaction.QueryRow(ctx, `
		INSERT INTO template_snapshots (day_template_id, user_id)
		VALUES ($1, $2)
		RETURNING id
	`, templateID, userID).Scan(&snapshotID)
	if err != nil {
		return err
	}

	for _, block := range blocks {
		_, err := transaction.Exec(ctx, `
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
