package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

type publicAcceptedEvent struct {
	ClientEventID string    `json:"client_event_id"`
	EventType     string    `json:"event_type"`
	CategoryID    *int      `json:"category_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type publicDayEventsResponse struct {
	publicDayRecord
	AcceptedEvents    []publicAcceptedEvent `json:"accepted_events"`
	DuplicateEventIDs []string              `json:"duplicate_client_event_ids"`
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

func (api *API) daysHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userID, authenticated := getUserID(request.Context())
	if !authenticated {
		http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
		return
	}
	path := strings.TrimPrefix(request.URL.Path, "/days")
	if path == "" || path == "/" {
		if request.Method != http.MethodGet {
			http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		api.getDays(responseWriter, request, userID)
		return
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 {
		if !isValidCalendarDate(parts[0]) {
			http.Error(responseWriter, "invalid date", http.StatusBadRequest)
			return
		}
		switch request.Method {
		case http.MethodGet:
			api.getDay(responseWriter, request, userID, parts[0])
		case http.MethodPost:
			api.createDay(responseWriter, request, userID, parts[0])
		default:
			http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 2 {
		if !isValidCalendarDate(parts[0]) {
			http.Error(responseWriter, "invalid date", http.StatusBadRequest)
			return
		}
		switch parts[1] {
		case "events":
			if request.Method == http.MethodPost {
				api.postDateEvents(responseWriter, request, userID, parts[0])
				return
			}
		case "blocks":
			if request.Method == http.MethodPut {
				api.putDateBlocks(responseWriter, request, userID, parts[0])
				return
			}
		case "template":
			if request.Method == http.MethodPut {
				api.putDateTemplate(responseWriter, request, userID, parts[0])
				return
			}
		}
	}
	http.Error(responseWriter, "not found", http.StatusNotFound)
}

func (api *API) getDays(responseWriter http.ResponseWriter, request *http.Request, userID int) {
	fromDate, toDate := request.URL.Query().Get("from"), request.URL.Query().Get("to")
	fromTime, fromError := time.Parse("2006-01-02", fromDate)
	toTime, toError := time.Parse("2006-01-02", toDate)
	if fromError != nil || toError != nil || toTime.Before(fromTime) {
		http.Error(responseWriter, "invalid date range", http.StatusBadRequest)
		return
	}
	records, err := api.dayRecordRepo.FindByDateRange(request.Context(), userID, fromDate, toDate)
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
	record, err := api.dayRecordRepo.FindByDate(request.Context(), userID, calendarDate)
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

func (api *API) createDay(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	record, err := api.dayRecordRepo.Create(request.Context(), userID, calendarDate)
	if errors.Is(err, ErrDayRecordAlreadyExists) {
		http.Error(responseWriter, "day record already exists", 409)
		return
	}
	if err != nil {
		HTTPError(responseWriter, request, api.logger, 500, "failed to create day", err, nil)
		return
	}
	responseWriter.WriteHeader(http.StatusCreated)
	writeJSON(responseWriter, toPublicDayRecord(record))
}

func (api *API) putDateBlocks(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	var input DayRecordBlocksInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(responseWriter, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validateActualBlocks(input.ActualBlocks); err != nil {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	if err := api.validateActualBlockCategoryIDs(request.Context(), userID, input.ActualBlocks); err != nil {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	record, err := api.dayRecordRepo.FindByDate(request.Context(), userID, calendarDate)
	if err != nil {
		http.Error(responseWriter, "day record not found", 404)
		return
	}
	if _, err = api.dayRecordRepo.ReplaceActualBlocks(request.Context(), record.ID, userID, input.ActualBlocks); err != nil {
		http.Error(responseWriter, "failed to replace actual blocks", 500)
		return
	}
	record, err = api.dayRecordRepo.FindByDate(request.Context(), userID, calendarDate)
	if err != nil {
		http.Error(responseWriter, "failed to fetch updated day", 500)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}

func (api *API) putDateTemplate(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	var input DayRecordTemplateInput
	if json.NewDecoder(request.Body).Decode(&input) != nil {
		http.Error(responseWriter, "invalid JSON", 400)
		return
	}
	record, err := api.dayRecordRepo.UpdateTemplateByDate(request.Context(), userID, calendarDate, input.DayTemplateID)
	if errors.Is(err, ErrDayRecordPast) {
		http.Error(responseWriter, err.Error(), 400)
		return
	}
	if err != nil {
		http.Error(responseWriter, "day record or template not found", 404)
		return
	}
	writeJSON(responseWriter, toPublicDayRecord(record))
}

func (api *API) postDateEvents(responseWriter http.ResponseWriter, request *http.Request, userID int, calendarDate string) {
	var input DayEventsInput
	if json.NewDecoder(request.Body).Decode(&input) != nil {
		http.Error(responseWriter, "invalid JSON", http.StatusBadRequest)
		return
	}
	if input.DeviceID <= 0 {
		http.Error(responseWriter, "device_id is required", http.StatusBadRequest)
		return
	}
	if err := validateDateEvents(input.Events); err != nil {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	if err := api.validateCategoryIDs(request.Context(), userID, input.Events); err != nil {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := api.dayRecordRepo.CreateEventsByDate(request.Context(), userID, calendarDate, input.DeviceID, input.Events)
	if errors.Is(err, ErrDeviceNotFound) {
		http.Error(responseWriter, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrAmendmentTargetNotFound) {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, ErrNonMonotonicTransitions) {
		http.Error(responseWriter, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		HTTPError(responseWriter, request, api.logger, http.StatusInternalServerError, "failed to append day events", err, nil)
		return
	}
	acceptedEvents := make([]publicAcceptedEvent, 0, len(result.AcceptedEvents))
	for _, event := range result.AcceptedEvents {
		clientEventID := ""
		if event.ClientEventID != nil {
			clientEventID = *event.ClientEventID
		}
		acceptedEvents = append(acceptedEvents, publicAcceptedEvent{
			ClientEventID: clientEventID,
			EventType:     event.EventType,
			CategoryID:    event.CategoryID,
			OccurredAt:    event.OccurredAt,
		})
	}
	response := toPublicDayRecord(result.Record)
	writeJSON(responseWriter, publicDayEventsResponse{
		publicDayRecord:   response,
		AcceptedEvents:    acceptedEvents,
		DuplicateEventIDs: result.DuplicateEventIDs,
	})
}

func writeJSON(responseWriter http.ResponseWriter, value interface{}) {
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(value)
}
