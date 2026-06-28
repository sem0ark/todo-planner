package main

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migration struct {
	ID   int
	Name string
	Up   func(ctx context.Context, db *pgxpool.Pool) error
	Down func(ctx context.Context, db *pgxpool.Pool) error
}

func ApplyMigrations(ctx context.Context, db *pgxpool.Pool, migrations []Migration) error {
	// Create migrations table
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	rows, err := db.Query(ctx, "SELECT id FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		applied[id] = true
	}

	// Sort migrations by ID
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	// Apply pending migrations
	for _, m := range migrations {
		if applied[m.ID] {
			continue
		}

		log.Printf("Applying migration %d: %s", m.ID, m.Name)

		if err := m.Up(ctx, db); err != nil {
			return fmt.Errorf("migration %d failed: %w", m.ID, err)
		}

		_, err := db.Exec(ctx,
			"INSERT INTO schema_migrations (id, name) VALUES ($1, $2)",
			m.ID, m.Name,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", m.ID, err)
		}
	}

	return nil
}
