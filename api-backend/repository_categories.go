package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// An activity category
type BlockCategory struct {
	ID             int             `json:"id"`
	UserID         int             `json:"user_id"`
	Name           string          `json:"name"`
	Color          string          `json:"color"` // hex color
	PomodoroConfig *PomodoroConfig `json:"pomodoro_config"`
	IsDeleted      bool            `json:"-"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) FindByUser(ctx context.Context, userID int) ([]BlockCategory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, color, pomodoro_config, created_at, updated_at
		FROM block_categories
		WHERE user_id = $1 AND is_deleted = FALSE
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]BlockCategory, 0, 10)
	for rows.Next() {
		var cat BlockCategory
		if err := rows.Scan(&cat.ID, &cat.UserID, &cat.Name, &cat.Color, &cat.PomodoroConfig, &cat.CreatedAt, &cat.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}

	return categories, rows.Err()
}

func (r *CategoryRepository) FindByID(ctx context.Context, id, userID int) (*BlockCategory, error) {
	var cat BlockCategory
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, name, color, pomodoro_config, is_deleted, created_at, updated_at
		FROM block_categories
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&cat.ID, &cat.UserID, &cat.Name, &cat.Color, &cat.PomodoroConfig, &cat.IsDeleted, &cat.CreatedAt, &cat.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *CategoryRepository) Create(ctx context.Context, input CategoryInput, userID int) (*BlockCategory, error) {
	now := time.Now()
	var cat BlockCategory

	err := r.db.QueryRow(ctx, `
		INSERT INTO block_categories (user_id, name, color, pomodoro_config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, name, color, pomodoro_config, created_at, updated_at
	`, userID, input.Name, input.Color, input.PomodoroConfig, now, now).Scan(
		&cat.ID,
		&cat.UserID,
		&cat.Name,
		&cat.Color,
		&cat.PomodoroConfig,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &cat, nil
}

func (r *CategoryRepository) Update(ctx context.Context, id int, input CategoryInput, userID int) (*BlockCategory, error) {
	now := time.Now()
	var cat BlockCategory

	err := r.db.QueryRow(ctx, `
		UPDATE block_categories
		SET name = $1, color = $2, pomodoro_config = $3, updated_at = $4
		WHERE id = $5 AND user_id = $6
		RETURNING id, user_id, name, color, pomodoro_config, created_at, updated_at
	`, input.Name, input.Color, input.PomodoroConfig, now, id, userID).Scan(
		&cat.ID,
		&cat.UserID,
		&cat.Name,
		&cat.Color,
		&cat.PomodoroConfig,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &cat, nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id, userID int) error {
	now := time.Now()
	result, err := r.db.Exec(ctx, `
		UPDATE block_categories
		SET is_deleted = TRUE, updated_at = $1
		WHERE id = $2 AND user_id = $3
	`, now, id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}

	return nil
}

var (
	ErrCategoryNotFound = &CategoryNotFoundError{}
)

type CategoryNotFoundError struct{}

func (e *CategoryNotFoundError) Error() string {
	return "category not found"
}
