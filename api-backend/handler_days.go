package main

import (
	"net/http"
	"time"
)

type PublicTimelineBlock struct {
	CategoryID      *int         `json:"category_id"`
	BlockType       string       `json:"block_type"`
	StartTime       ScheduleTime `json:"start_time"`
	DurationMinutes int          `json:"duration_minutes"`
}

type PublicDayRecord struct {
	CalendarDate          CalendarDate          `json:"calendar_date"`
	DayTemplateID         *int                  `json:"day_template_id"`
	TimezoneOffsetMinutes *int                  `json:"timezone_offset_minutes"`
	TimezoneOffsetLocked  bool                  `json:"timezone_offset_locked"`
	Plan                  []PublicTemplateBlock `json:"plan"`
	Actual                []PublicTimelineBlock `json:"actual"`
	CreatedAt             APITimestamp          `json:"created_at"`
	UpdatedAt             APITimestamp          `json:"updated_at"`
}

type PublicDayRangeEntry struct {
	CalendarDate CalendarDate     `json:"calendar_date"`
	DayRecord    *PublicDayRecord `json:"day_record"`
}

type PublicDayRangeResponse struct {
	Days []PublicDayRangeEntry `json:"days"`
}

func toPublicDayRecord(record *DayRecord) PublicDayRecord {
	plan := toPublicTemplateBlocks(record.SnapshotBlocks)
	actualBlocks := make([]PublicTimelineBlock, 0, len(record.ActualBlocks))
	for _, block := range record.ActualBlocks {
		startTime := block.StartTime
		if record.TimezoneOffsetMinutes != nil {
			startTime = shiftScheduleTime(startTime, *record.TimezoneOffsetMinutes)
		}
		actualBlocks = append(actualBlocks, PublicTimelineBlock{CategoryID: block.CategoryID,
			BlockType: block.BlockType, StartTime: startTime,
			DurationMinutes: block.DurationMinutes})
	}
	return PublicDayRecord{CalendarDate: record.CalendarDate, DayTemplateID: record.DayTemplateID,
		TimezoneOffsetMinutes: record.TimezoneOffsetMinutes, TimezoneOffsetLocked: record.TimezoneOffsetLocked,
		Plan: plan, Actual: actualBlocks, CreatedAt: APITimestamp(record.CreatedAt),
		UpdatedAt: APITimestamp(record.UpdatedAt)}
}

func shiftScheduleTime(value ScheduleTime, offsetMinutes int) ScheduleTime {
	return ScheduleTime(time.Time(value).Add(time.Duration(offsetMinutes) * time.Minute))
}

func (api *API) getDays(responseWriter http.ResponseWriter, request *http.Request, userID int) {
	fromDate, fromError := parseCalendarDate(request.URL.Query().Get("from"))
	toDate, toError := parseCalendarDate(request.URL.Query().Get("to"))
	if fromError != nil || toError != nil || toDate.Before(fromDate) {
		http.Error(responseWriter, ErrInvalidDayDateRange.Error(), http.StatusBadRequest)
		return
	}
	records, err := api.dayRecordRepo.FindDateRange(request.Context(), userID, fromDate, toDate)
	if err != nil {
		HTTPError(responseWriter, request, api.logger, 500, "failed to fetch days", err, nil)
		return
	}
	publicEntries := make([]PublicDayRangeEntry, 0, len(records))
	for _, entry := range records {
		var publicRecord *PublicDayRecord
		if entry.Record != nil {
			convertedRecord := toPublicDayRecord(entry.Record)
			publicRecord = &convertedRecord
		}
		publicEntries = append(publicEntries, PublicDayRangeEntry{
			CalendarDate: entry.CalendarDate,
			DayRecord:    publicRecord,
		})
	}
	writeJSON(responseWriter, PublicDayRangeResponse{Days: publicEntries})
}

func (api *API) getDay(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	parsedDate, err := parseCalendarDate(calendarDate)
	if err != nil {
		http.Error(responseWriter, "invalid date", http.StatusBadRequest)
		return
	}
	record, err := api.dayRecordRepo.FindByDate(request.Context(), userID, parsedDate)
	if err != nil {
		if writeAppError(responseWriter, err) {
			return
		}
		HTTPError(responseWriter, request, api.logger, 500, "failed to fetch day", err, nil)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}
