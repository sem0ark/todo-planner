package main

import "time"

type TemplateSnapshot struct {
	ID             int             `json:"id"`
	DayTemplateID  int             `json:"day_template_id"`
	UserID         int             `json:"user_id"`
	SnapshotBlocks []SnapshotBlock `json:"snapshot_blocks,omitempty"`
	SnapshottedAt  time.Time       `json:"snapshotted_at"`
}

// A time block in a template snapshot
type SnapshotBlock struct {
	ID              int    `json:"id"`
	SnapshotID      int    `json:"snapshot_id,omitempty"`
	CategoryID      int    `json:"category_id"`
	StartTime       string `json:"start_time"` // HH:MM:SS
	DurationMinutes int    `json:"duration_minutes"`
}

// A day's tracked activity
type DayRecord struct {
	ID             int             `json:"id"`
	UserID         int             `json:"user_id"`
	SnapshotID     *int            `json:"snapshot_id,omitempty"`
	CalendarDate   string          `json:"calendar_date"` // YYYY-MM-DD
	ReviewStatus   string          `json:"review_status"` // Unreviewed | Reviewed | Ignored
	SnapshotBlocks []SnapshotBlock `json:"snapshot_blocks,omitempty"`
	ActualBlocks   []ActualBlock   `json:"actual_blocks,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// A real-time activity confirmation or transition
type DayEvent struct {
	ID                 int       `json:"id"`
	DayRecordID        int       `json:"day_record_id,omitempty"`
	EventType          string    `json:"event_type"` // confirmation | transition
	OutgoingCategoryID *int      `json:"outgoing_category_id,omitempty"`
	IncomingCategoryID *int      `json:"incoming_category_id,omitempty"`
	OccurredAt         time.Time `json:"occurred_at"`
}

// A manual edit to a day's record
type RetroactiveEdit struct {
	ID              int       `json:"id"`
	DayRecordID     int       `json:"day_record_id,omitempty"`
	EditType        string    `json:"edit_type"` // resize | move | relabel | split | mark_blank
	CategoryID      *int      `json:"category_id,omitempty"`
	BlockStart      string    `json:"block_start"` // HH:MM:SS
	DurationMinutes *int      `json:"duration_minutes,omitempty"`
	OccurredAt      time.Time `json:"occurred_at"`
}

// A derived time block for a day record
type ActualBlock struct {
	ID              int       `json:"id"`
	DayRecordID     int       `json:"day_record_id,omitempty"`
	CategoryID      *int      `json:"category_id,omitempty"`
	BlockType       string    `json:"block_type"` // actual | blank | untracked
	StartTime       string    `json:"start_time"` // HH:MM:SS
	DurationMinutes int       `json:"duration_minutes"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Weekly schedule update request data
type WeeklyScheduleInput struct {
	WeeklySchedule []WeeklyScheduleEntry `json:"weekly_schedule"`
}

// A single day's template assignment
type WeeklyScheduleEntry struct {
	DayOfWeek     int  `json:"day_of_week"`
	DayTemplateID *int `json:"day_template_id,omitempty"`
}

// Schedule override creation/update request data
type ScheduleOverrideInput struct {
	DayTemplateID *int `json:"day_template_id,omitempty"`
}

// Day record creation request data
type DayRecordInput struct {
	CalendarDate string `json:"calendar_date"` // YYYY-MM-DD
}

// Day record status update request data
type DayRecordStatusInput struct {
	ReviewStatus string `json:"review_status"` // Reviewed | Ignored
}

// Batch day event submission request data
type DayEventsInput struct {
	Events []DayEventInput `json:"events"`
}

// A single day event input
type DayEventInput struct {
	EventType          string    `json:"event_type"` // confirmation | transition
	OutgoingCategoryID *int      `json:"outgoing_category_id,omitempty"`
	IncomingCategoryID *int      `json:"incoming_category_id,omitempty"`
	OccurredAt         time.Time `json:"occurred_at"`
}

// Batch retroactive edit submission request data
type RetroactiveEditsInput struct {
	Edits []RetroactiveEditInput `json:"edits"`
}

// A single retroactive edit input
type RetroactiveEditInput struct {
	EditType        string    `json:"edit_type"` // resize | move | relabel | split | mark_blank
	CategoryID      *int      `json:"category_id,omitempty"`
	BlockStart      string    `json:"block_start"` // ISO 8601 time string
	DurationMinutes *int      `json:"duration_minutes,omitempty"`
	OccurredAt      time.Time `json:"occurred_at"`
}

// Sync request
type SyncRequest struct {
	DeviceID   int              `json:"device_id"`
	LastSyncAt *time.Time       `json:"last_sync_at"`
	Changes    []ChangeLogEntry `json:"changes"`
}

// Sync response
type SyncResponse struct {
	SyncedAt time.Time        `json:"synced_at"`
	Changes  []ChangeLogEntry `json:"changes"`
}

// Change log entry
type ChangeLogEntry struct {
	EntityType string    `json:"entity_type"` // category | template_group | day_template | weekly_schedule | schedule_override | day_record | settings
	EntityID   int       `json:"entity_id"`
	Operation  string    `json:"operation"` // create | update | delete
	OccurredAt time.Time `json:"occurred_at"`
}
