package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetMigrations() []Migration {
	return []Migration{
		{
			ID:   1,
			Name: "create_todos_table",
			Up: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					CREATE TABLE todos (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						description TEXT,
						completed BOOLEAN DEFAULT FALSE,
						priority VARCHAR(10) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
						due_date TIMESTAMP,
						created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
					);
				`)
				return err
			},
			Down: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, "DROP TABLE IF EXISTS todos;")
				return err
			},
		},
		{
			ID:   2,
			Name: "create_todos_indexes",
			Up: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					CREATE INDEX idx_todos_completed ON todos(completed);
					CREATE INDEX idx_todos_priority ON todos(priority);
					CREATE INDEX idx_todos_due_date ON todos(due_date);
				`)
				return err
			},
			Down: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					DROP INDEX IF EXISTS idx_todos_completed;
					DROP INDEX IF EXISTS idx_todos_priority;
					DROP INDEX IF EXISTS idx_todos_due_date;
				`)
				return err
			},
		},
		{
			ID:   3,
			Name: "seed_initial_todos",
			Up: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					INSERT INTO todos (title, description, completed, priority, due_date) VALUES
					('Setup project', 'Initialize git repository and basic structure', true, 'high', NOW() + INTERVAL '1 day'),
					('Implement API', 'Create basic handlers for CRUD operations', false, 'medium', NOW() + INTERVAL '3 days'),
					('Add migrations', 'Setup migration system and seed data', false, 'high', NOW() + INTERVAL '2 days'),
					('Write tests', 'Implement unit and integration tests', false, 'low', NOW() + INTERVAL '7 days');
				`)
				return err
			},
			Down: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, "DELETE FROM todos;")
				return err
			},
		},
	}
}
