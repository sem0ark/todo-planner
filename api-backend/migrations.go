package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetMigrations() []Migration {
	return []Migration{
		{
			ID:   1,
			Name: "reset_schema",
			Up: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					...
				`)
				return err
			},
			Down: func(ctx context.Context, db *pgxpool.Pool) error {
				return nil
			},
		},
	}
}

