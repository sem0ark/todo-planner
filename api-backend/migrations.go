package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetMigrations() []Migration {
	return []Migration{
		{
			ID:   1,
			Name: "create_initial_schema",
			Up: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					-- Users table
					CREATE TABLE IF NOT EXISTS users (
						id SERIAL PRIMARY KEY,
						username VARCHAR(255) UNIQUE NOT NULL,
						password_hash VARCHAR(255) NOT NULL,
						created_at TIMESTAMPTZ NOT NULL DEFAULT now()
					);

					-- User settings table
					CREATE TABLE IF NOT EXISTS user_settings (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						day_boundary_time TIME NOT NULL DEFAULT '04:00:00',
						updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						UNIQUE(user_id)
					);

					-- Devices table
					CREATE TABLE IF NOT EXISTS devices (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						platform VARCHAR(50) NOT NULL,
						token_hash VARCHAR(255) NOT NULL,
						registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						last_sync_at TIMESTAMPTZ
					);

					-- Change log table
					CREATE TABLE IF NOT EXISTS change_log (
						id SERIAL PRIMARY KEY,
						device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						entity_type VARCHAR(50) NOT NULL,
						entity_id INTEGER NOT NULL,
						operation VARCHAR(20) NOT NULL,
						occurred_at TIMESTAMPTZ NOT NULL
					);

					-- Block categories table
					CREATE TABLE IF NOT EXISTS block_categories (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						name VARCHAR(255) NOT NULL,
						color VARCHAR(7) NOT NULL,
						is_deleted BOOLEAN DEFAULT FALSE,
						created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
					);

					-- Template groups table
					CREATE TABLE IF NOT EXISTS template_groups (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						name VARCHAR(255) NOT NULL,
						is_deleted BOOLEAN DEFAULT FALSE,
						created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
					);

					-- Day templates table
					CREATE TABLE IF NOT EXISTS day_templates (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						template_group_id INTEGER REFERENCES template_groups(id) ON DELETE SET NULL,
						name VARCHAR(255) NOT NULL,
						is_deleted BOOLEAN DEFAULT FALSE,
						created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
					);

					-- Planned blocks table
					CREATE TABLE IF NOT EXISTS planned_blocks (
						id SERIAL PRIMARY KEY,
						day_template_id INTEGER NOT NULL REFERENCES day_templates(id) ON DELETE CASCADE,
						category_id INTEGER NOT NULL REFERENCES block_categories(id) ON DELETE CASCADE,
						start_time TIME NOT NULL,
						duration_minutes INTEGER NOT NULL
					);

					-- Template snapshots table
					CREATE TABLE IF NOT EXISTS template_snapshots (
						id SERIAL PRIMARY KEY,
						day_template_id INTEGER NOT NULL REFERENCES day_templates(id) ON DELETE CASCADE,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						snapshotted_at TIMESTAMPTZ NOT NULL DEFAULT now()
					);

					-- Snapshot blocks table
					CREATE TABLE IF NOT EXISTS snapshot_blocks (
						id SERIAL PRIMARY KEY,
						snapshot_id INTEGER NOT NULL REFERENCES template_snapshots(id) ON DELETE CASCADE,
						category_id INTEGER NOT NULL REFERENCES block_categories(id) ON DELETE CASCADE,
						start_time TIME NOT NULL,
						duration_minutes INTEGER NOT NULL
					);

					-- Weekly schedule table
					CREATE TABLE IF NOT EXISTS weekly_schedule (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						day_of_week INTEGER NOT NULL,
						day_template_id INTEGER REFERENCES day_templates(id) ON DELETE SET NULL,
						updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						UNIQUE(user_id, day_of_week)
					);

					-- Schedule overrides table
					CREATE TABLE IF NOT EXISTS schedule_overrides (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						calendar_date DATE NOT NULL,
						day_template_id INTEGER REFERENCES day_templates(id) ON DELETE SET NULL,
						created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						UNIQUE(user_id, calendar_date)
					);

					-- Day records table
					CREATE TABLE IF NOT EXISTS day_records (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						snapshot_id INTEGER REFERENCES template_snapshots(id) ON DELETE SET NULL,
						calendar_date DATE NOT NULL,
						review_status VARCHAR(20) NOT NULL DEFAULT 'Unreviewed',
						created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
						UNIQUE(user_id, calendar_date)
					);

					-- Day events table
					CREATE TABLE IF NOT EXISTS day_events (
						id SERIAL PRIMARY KEY,
						day_record_id INTEGER NOT NULL REFERENCES day_records(id) ON DELETE CASCADE,
						event_type VARCHAR(20) NOT NULL,
						outgoing_category_id INTEGER REFERENCES block_categories(id) ON DELETE CASCADE,
						incoming_category_id INTEGER REFERENCES block_categories(id) ON DELETE CASCADE,
						occurred_at TIMESTAMPTZ NOT NULL
					);

					-- Retroactive edits table
					CREATE TABLE IF NOT EXISTS retroactive_edits (
						id SERIAL PRIMARY KEY,
						day_record_id INTEGER NOT NULL REFERENCES day_records(id) ON DELETE CASCADE,
						edit_type VARCHAR(20) NOT NULL,
						category_id INTEGER REFERENCES block_categories(id) ON DELETE CASCADE,
						block_start TIME NOT NULL,
						duration_minutes INTEGER,
						occurred_at TIMESTAMPTZ NOT NULL
					);

					-- Actual blocks table
					CREATE TABLE IF NOT EXISTS actual_blocks (
						id SERIAL PRIMARY KEY,
						day_record_id INTEGER NOT NULL REFERENCES day_records(id) ON DELETE CASCADE,
						category_id INTEGER REFERENCES block_categories(id) ON DELETE CASCADE,
						block_type VARCHAR(20) NOT NULL,
						start_time TIME NOT NULL,
						duration_minutes INTEGER NOT NULL,
						updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
					);
				`)
				return err
			},
			Down: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					DROP TABLE IF EXISTS actual_blocks CASCADE;
					DROP TABLE IF EXISTS retroactive_edits CASCADE;
					DROP TABLE IF EXISTS day_events CASCADE;
					DROP TABLE IF EXISTS day_records CASCADE;
					DROP TABLE IF EXISTS schedule_overrides CASCADE;
					DROP TABLE IF EXISTS weekly_schedule CASCADE;
					DROP TABLE IF EXISTS snapshot_blocks CASCADE;
					DROP TABLE IF EXISTS template_snapshots CASCADE;
					DROP TABLE IF EXISTS planned_blocks CASCADE;
					DROP TABLE IF EXISTS day_templates CASCADE;
					DROP TABLE IF EXISTS template_groups CASCADE;
					DROP TABLE IF EXISTS block_categories CASCADE;
					DROP TABLE IF EXISTS change_log CASCADE;
					DROP TABLE IF EXISTS devices CASCADE;
					DROP TABLE IF EXISTS user_settings CASCADE;
					DROP TABLE IF EXISTS users CASCADE;
				`)
				return err
			},
		},
		{
			ID:   2,
			Name: "create_indexes",
			Up: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					-- Change log indexes
					CREATE INDEX IF NOT EXISTS idx_change_log_user_occurred ON change_log(user_id, occurred_at);
					CREATE INDEX IF NOT EXISTS idx_change_log_device_occurred ON change_log(device_id, occurred_at);

					-- Block categories indexes
					CREATE INDEX IF NOT EXISTS idx_block_categories_user_deleted ON block_categories(user_id, is_deleted);

					-- Template groups indexes
					CREATE INDEX IF NOT EXISTS idx_template_groups_user_deleted ON template_groups(user_id, is_deleted);

					-- Day templates indexes
					CREATE INDEX IF NOT EXISTS idx_day_templates_user_deleted ON day_templates(user_id, is_deleted);
					CREATE INDEX IF NOT EXISTS idx_day_templates_group ON day_templates(template_group_id);

					-- Planned blocks indexes
					CREATE INDEX IF NOT EXISTS idx_planned_blocks_template ON planned_blocks(day_template_id);

					-- Template snapshots indexes
					CREATE INDEX IF NOT EXISTS idx_template_snapshots_template_time ON template_snapshots(day_template_id, snapshotted_at);

					-- Snapshot blocks indexes
					CREATE INDEX IF NOT EXISTS idx_snapshot_blocks_snapshot ON snapshot_blocks(snapshot_id);

					-- Weekly schedule indexes
					CREATE INDEX IF NOT EXISTS idx_weekly_schedule_user_dow ON weekly_schedule(user_id, day_of_week);

					-- Schedule overrides indexes
					CREATE INDEX IF NOT EXISTS idx_schedule_overrides_user_date ON schedule_overrides(user_id, calendar_date);

					-- Day records indexes
					CREATE INDEX IF NOT EXISTS idx_day_records_user_date ON day_records(user_id, calendar_date);
					CREATE INDEX IF NOT EXISTS idx_day_records_user_status ON day_records(user_id, review_status);

					-- Day events indexes
					CREATE INDEX IF NOT EXISTS idx_day_events_record_occurred ON day_events(day_record_id, occurred_at);

					-- Retroactive edits indexes
					CREATE INDEX IF NOT EXISTS idx_retroactive_edits_record_occurred ON retroactive_edits(day_record_id, occurred_at);

					-- Actual blocks indexes
					CREATE INDEX IF NOT EXISTS idx_actual_blocks_record ON actual_blocks(day_record_id);
					CREATE INDEX IF NOT EXISTS idx_actual_blocks_record_start ON actual_blocks(day_record_id, start_time);
				`)
				return err
			},
			Down: func(ctx context.Context, db *pgxpool.Pool) error {
				_, err := db.Exec(ctx, `
					DROP INDEX IF EXISTS idx_actual_blocks_record_start;
					DROP INDEX IF EXISTS idx_actual_blocks_record;
					DROP INDEX IF EXISTS idx_retroactive_edits_record_occurred;
					DROP INDEX IF EXISTS idx_day_events_record_occurred;
					DROP INDEX IF EXISTS idx_day_records_user_status;
					DROP INDEX IF EXISTS idx_day_records_user_date;
					DROP INDEX IF EXISTS idx_schedule_overrides_user_date;
					DROP INDEX IF EXISTS idx_weekly_schedule_user_dow;
					DROP INDEX IF EXISTS idx_snapshot_blocks_snapshot;
					DROP INDEX IF EXISTS idx_template_snapshots_template_time;
					DROP INDEX IF EXISTS idx_planned_blocks_template;
					DROP INDEX IF EXISTS idx_day_templates_group;
					DROP INDEX IF EXISTS idx_day_templates_user_deleted;
					DROP INDEX IF EXISTS idx_template_groups_user_deleted;
					DROP INDEX IF EXISTS idx_block_categories_user_deleted;
					DROP INDEX IF EXISTS idx_change_log_device_occurred;
					DROP INDEX IF EXISTS idx_change_log_user_occurred;
				`)
				return err
			},
		},
	}
}
