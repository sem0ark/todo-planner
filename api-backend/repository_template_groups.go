package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTemplateGroupNotFound = errors.New("template group not found")

// A grouping for day templates
type TemplateGroup struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	IsDeleted bool      `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Template group creation/update request data
type TemplateGroupInput struct {
	Name string `json:"name"`
}

type TemplateGroupRepository struct {
	db *pgxpool.Pool
}

func NewTemplateGroupRepository(db *pgxpool.Pool) *TemplateGroupRepository {
	return &TemplateGroupRepository{db: db}
}

func (r *TemplateGroupRepository) FindByUser(ctx context.Context, userID int) ([]TemplateGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, created_at, updated_at
		FROM template_groups
		WHERE user_id = $1 AND is_deleted = FALSE
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]TemplateGroup, 0)
	for rows.Next() {
		var group TemplateGroup
		if err := rows.Scan(&group.ID, &group.UserID, &group.Name, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

func (r *TemplateGroupRepository) FindByID(ctx context.Context, id, userID int) (*TemplateGroup, error) {
	var group TemplateGroup
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, name, is_deleted, created_at, updated_at
		FROM template_groups
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&group.ID, &group.UserID, &group.Name, &group.IsDeleted, &group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &group, nil
}

func (r *TemplateGroupRepository) Create(ctx context.Context, input TemplateGroupInput, userID int) (*TemplateGroup, error) {
	var group TemplateGroup
	err := r.db.QueryRow(ctx, `
		INSERT INTO template_groups (user_id, name, is_deleted, created_at, updated_at)
		VALUES ($1, $2, FALSE, NOW(), NOW())
		RETURNING id, user_id, name, created_at, updated_at
	`, userID, input.Name).Scan(&group.ID, &group.UserID, &group.Name, &group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &group, nil
}

func (r *TemplateGroupRepository) Update(ctx context.Context, id int, input TemplateGroupInput, userID int) (*TemplateGroup, error) {
	var group TemplateGroup
	err := r.db.QueryRow(ctx, `
		UPDATE template_groups
		SET name = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3 AND is_deleted = FALSE
		RETURNING id, user_id, name, created_at, updated_at
	`, input.Name, id, userID).Scan(&group.ID, &group.UserID, &group.Name, &group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		return nil, ErrTemplateGroupNotFound
	}

	return &group, nil
}

func (r *TemplateGroupRepository) Delete(ctx context.Context, id, userID int) error {
	result, err := r.db.Exec(ctx, `
		UPDATE template_groups
		SET is_deleted = TRUE, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`, id, userID)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrTemplateGroupNotFound
	}

	return nil
}
