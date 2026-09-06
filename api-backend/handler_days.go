package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type publicSnapshotBlock struct {
	CategoryID      int    `json:"category_id"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
}

type publicSnapshot struct {
	SnapshottedAt time.Time             `json:"snapshotted_at"`
	Blocks        []publicSnapshotBlock `json:"blocks"`
}

type publicActualBlock struct {
	CategoryID      *int   `json:"category_id"`
	BlockType       string `json:"block_type"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
	IsOpen          bool   `json:"is_open"`
}

type publicDayRecord struct {
	CalendarDate  string              `json:"calendar_date"`
	DayTemplateID *int                `json:"day_template_id"`
	Snapshot      *publicSnapshot     `json:"snapshot"`
	ActualBlocks  []publicActualBlock `json:"actual_blocks"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

type publicDayRecordsResponse struct {
	DayRecords []publicDayRecord `json:"day_records"`
}

func toPublicDayRecord(record *DayRecord) publicDayRecord {
	var snapshot *publicSnapshot
	if record.SnapshotID != nil {
		blocks := make([]publicSnapshotBlock, 0, len(record.SnapshotBlocks))
		for _, block := range record.SnapshotBlocks {
			blocks = append(blocks, publicSnapshotBlock{block.CategoryID, shortClock(block.StartTime), block.DurationMinutes})
		}
		snapshot = &publicSnapshot{record.SnapshottedAt, blocks}
	}
	actualBlocks := make([]publicActualBlock, 0, len(record.ActualBlocks))
	for _, block := range record.ActualBlocks {
		actualBlocks = append(actualBlocks, publicActualBlock{block.CategoryID, block.BlockType, shortClock(block.StartTime), block.DurationMinutes, block.IsOpen})
	}
	return publicDayRecord{record.CalendarDate, record.DayTemplateID, snapshot, actualBlocks, record.CreatedAt, record.UpdatedAt}
}

func shortClock(value string) string {
	if len(value) > 5 {
		return value[:5]
	}
	return value
}

func (api *API) getDays(responseWriter http.ResponseWriter, request *http.Request, userID int) {
	records, err := api.dayService.GetDays(request.Context(), userID, request.URL.Query().Get("from"), request.URL.Query().Get("to"))
	if errors.Is(err, ErrInvalidDayDateRange) {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		HTTPError(responseWriter, request, api.logger, 500, "failed to fetch days", err, nil)
		return
	}
	publicRecords := make([]publicDayRecord, 0, len(records))
	for index := range records {
		publicRecords = append(publicRecords, toPublicDayRecord(&records[index]))
	}
	writeJSON(responseWriter, publicDayRecordsResponse{DayRecords: publicRecords})
}

func (api *API) getDay(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	record, err := api.dayService.GetDay(request.Context(), userID, calendarDate)
	if errors.Is(err, ErrDayRecordNotFound) {
		http.Error(responseWriter, "day record not found", 404)
		return
	}
	if err != nil {
		HTTPError(responseWriter, request, api.logger, 500, "failed to fetch day", err, nil)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}

func writeJSON(responseWriter http.ResponseWriter, value interface{}) {
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(value)
}
