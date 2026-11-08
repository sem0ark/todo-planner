package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// setupIsolatedMigrationDB creates an empty scratch database so that a partial
// migration list can be applied without touching the shared test database.
func setupIsolatedMigrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration tests")
	}
	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, testDBURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	databaseName := fmt.Sprintf("migration_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+databaseName); err != nil {
		adminPool.Close()
		t.Fatalf("failed to create scratch database: %v", err)
	}
	poolConfig, err := pgxpool.ParseConfig(testDBURL)
	if err != nil {
		t.Fatalf("failed to parse test database URL: %v", err)
	}
	poolConfig.ConnConfig.Database = databaseName
	scratchPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("failed to connect to scratch database: %v", err)
	}
	t.Cleanup(func() {
		scratchPool.Close()
		_, _ = adminPool.Exec(ctx, "DROP DATABASE IF EXISTS "+databaseName+" WITH (FORCE)")
		adminPool.Close()
	})
	return scratchPool
}

func TestMigration12ConvertsActualBlocksAndDayOffsets(t *testing.T) {
	ctx := context.Background()
	database := setupIsolatedMigrationDB(t)
	var before, all []Migration
	for _, migration := range GetMigrations() {
		all = append(all, migration)
		if migration.ID < 12 {
			before = append(before, migration)
		}
	}
	if len(before) != len(all)-1 {
		t.Fatalf("expected migration 12 to be the last migration, got %d of %d", len(before), len(all))
	}
	if err := ApplyMigrations(ctx, database, before); err != nil {
		t.Fatalf("failed to apply migrations 1..11: %v", err)
	}

	var userID, lockedRecordID, unlockedRecordID, noOffsetRecordID int
	mustScan := func(query string, destination *int, arguments ...any) {
		t.Helper()
		if err := database.QueryRow(ctx, query, arguments...).Scan(destination); err != nil {
			t.Fatalf("setup query failed: %v", err)
		}
	}
	mustScan(`INSERT INTO users (username, password_hash) VALUES ('migration-user', 'x') RETURNING id`, &userID)
	mustScan(`INSERT INTO day_records (user_id, calendar_date, timezone_offset_minutes, timezone_offset_locked)
		VALUES ($1, '2026-10-19', 120, TRUE) RETURNING id`, &lockedRecordID, userID)
	// Unlocked day: the offset is recovered from the first event's local timestamp.
	mustScan(`INSERT INTO day_records (user_id, calendar_date) VALUES ($1, '2026-10-20') RETURNING id`, &unlockedRecordID, userID)
	mustScan(`INSERT INTO day_records (user_id, calendar_date) VALUES ($1, '2026-10-21') RETURNING id`, &noOffsetRecordID, userID)
	setupStatements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO day_events (day_record_id, event_type, occurred_at, occurred_at_local)
			VALUES ($1, 'note', '2026-10-19T08:00:00Z', '2026-10-19T10:00:00+02:00')`, []any{lockedRecordID}},
		{`INSERT INTO day_events (day_record_id, event_type, occurred_at, occurred_at_local)
			VALUES ($1, 'note', '2026-10-20T08:00:00Z', '2026-10-20T03:00:00-05:00')`, []any{unlockedRecordID}},
		{`INSERT INTO actual_blocks (day_record_id, block_type, start_time, duration_minutes)
			VALUES ($1, 'blank', '06:30:00', 30)`, []any{lockedRecordID}},
		{`INSERT INTO actual_blocks (day_record_id, block_type, start_time, duration_minutes)
			VALUES ($1, 'blank', '22:30:00', 30)`, []any{lockedRecordID}},
	}
	for _, statement := range setupStatements {
		if _, err := database.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("setup insert failed: %v", err)
		}
	}

	if err := ApplyMigrations(ctx, database, all); err != nil {
		t.Fatalf("migration 12 failed: %v", err)
	}

	rows, err := database.Query(ctx, `SELECT start_at FROM actual_blocks ORDER BY id`)
	if err != nil {
		t.Fatalf("failed to read converted blocks: %v", err)
	}
	defer rows.Close()
	var starts []time.Time
	for rows.Next() {
		var startAt time.Time
		if err := rows.Scan(&startAt); err != nil {
			t.Fatal(err)
		}
		starts = append(starts, startAt.UTC())
	}
	expected := []time.Time{parseTime("2026-10-19T06:30:00Z"), parseTime("2026-10-18T22:30:00Z")}
	if len(starts) != 2 || !starts[0].Equal(expected[0]) || !starts[1].Equal(expected[1]) {
		t.Fatalf("expected start_at %v, got %v", expected, starts)
	}

	offsets := map[int]int{lockedRecordID: 120, unlockedRecordID: -300, noOffsetRecordID: 0}
	for recordID, expectedOffset := range offsets {
		var offset int
		if err := database.QueryRow(ctx, `SELECT timezone_offset_minutes FROM day_records WHERE id = $1`, recordID).Scan(&offset); err != nil {
			t.Fatal(err)
		}
		if offset != expectedOffset {
			t.Errorf("record %d: expected offset %d, got %d", recordID, expectedOffset, offset)
		}
	}

	var removedColumns int
	if err := database.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND (
			(table_name = 'actual_blocks' AND column_name = 'start_time') OR
			(table_name = 'day_records' AND column_name = 'timezone_offset_locked') OR
			(table_name = 'day_events' AND column_name IN ('occurred_at_local', 'corrected_at_local')))
	`).Scan(&removedColumns); err != nil {
		t.Fatal(err)
	}
	if removedColumns != 0 {
		t.Errorf("expected old columns to be dropped, %d remain", removedColumns)
	}
	var nullable string
	if err := database.QueryRow(ctx, `
		SELECT is_nullable FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'day_records' AND column_name = 'timezone_offset_minutes'
	`).Scan(&nullable); err != nil || nullable != "NO" {
		t.Errorf("expected timezone_offset_minutes NOT NULL, got %q (%v)", nullable, err)
	}
	var blocksEdited bool
	if err := database.QueryRow(ctx, `SELECT blocks_edited FROM day_records WHERE id = $1`, lockedRecordID).Scan(&blocksEdited); err != nil {
		t.Fatal(err)
	}
	if !blocksEdited {
		t.Error("expected a past day with stored blocks to be frozen (blocks_edited)")
	}
}
