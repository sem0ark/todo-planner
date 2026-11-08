package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAmendmentTargetNotFound = NewBadRequestError("amendment target event not found")
)

// DayTrackingService coordinates event ingestion and actual-block updates.
type DayTrackingService struct {
	db              *pgxpool.Pool
	dayRecordRepo   *DayRecordRepository
	dayEventRepo    *DayEventRepository
	actualBlockRepo *ActualBlockRepository
	dayTemplateRepo *DayTemplateRepository
	scheduleRepo    *ScheduleRepository
}

type DayRangeEntry struct {
	CalendarDate CalendarDate
	Record       *DayRecord
}

type DateEventResult struct {
	Record            *DayRecord
	AcceptedEvents    []DayEvent
	DuplicateEventIDs []string
}

func NewDayTrackingService(
	db *pgxpool.Pool,
	dayRecordRepo *DayRecordRepository,
	dayEventRepo *DayEventRepository,
	actualBlockRepo *ActualBlockRepository,
	dayTemplateRepo *DayTemplateRepository,
	scheduleRepo *ScheduleRepository,
) *DayTrackingService {
	return &DayTrackingService{
		db:              db,
		dayRecordRepo:   dayRecordRepo,
		dayEventRepo:    dayEventRepo,
		actualBlockRepo: actualBlockRepo,
		dayTemplateRepo: dayTemplateRepo,
		scheduleRepo:    scheduleRepo,
	}
}

// CreateEventsByDate persists a retry-safe batch for an existing day record and,
// unless the day's blocks were manually edited, rebuilds them from all events.
func (service *DayTrackingService) CreateEventsByDate(
	ctx context.Context,
	userID int,
	calendarDate CalendarDate,
	deviceID int,
	inputs []DayEventInput,
) (*DateEventResult, error) {
	transaction, err := service.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	deviceUserID, err := service.dayEventRepo.FindDeviceUserID(ctx, transaction, deviceID)
	if err != nil {
		return nil, err
	}
	if deviceUserID != userID {
		return nil, ErrDeviceNotFound
	}
	dayRecordID, err := service.dayRecordRepo.FindIDForDate(ctx, transaction, userID, calendarDate)
	if err != nil {
		return nil, err
	}

	existingEventIDs, err := service.dayEventRepo.FindExistingClientEventIDs(
		ctx,
		transaction,
		dayRecordID,
		eventLookupIDs(inputs),
	)
	if err != nil {
		return nil, err
	}
	newInputs, duplicateEventIDs, err := prepareNewDateEventInputs(inputs, existingEventIDs)
	if err != nil {
		return nil, err
	}
	result := &DateEventResult{DuplicateEventIDs: duplicateEventIDs}
	var insertDuplicateIDs []string
	result.AcceptedEvents, insertDuplicateIDs, err = service.dayEventRepo.Insert(
		ctx,
		transaction,
		dayRecordID,
		deviceID,
		newInputs,
	)
	if err != nil {
		return nil, err
	}
	result.DuplicateEventIDs = append(result.DuplicateEventIDs, insertDuplicateIDs...)

	result.Record, err = service.dayRecordRepo.TouchAndFindByID(ctx, transaction, dayRecordID, userID)
	if err != nil {
		return nil, err
	}
	if !result.Record.BlocksEdited {
		if err := service.rebuildActualBlocksTx(ctx, transaction, dayRecordID); err != nil {
			return nil, err
		}
	}
	if err := service.hydrateRecord(ctx, transaction, result.Record); err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (service *DayTrackingService) FindByDate(
	ctx context.Context,
	userID int,
	calendarDate CalendarDate,
) (*DayRecord, error) {
	record, err := service.dayRecordRepo.FindByDate(ctx, userID, calendarDate)
	if err != nil {
		return nil, err
	}
	if err := service.hydrateRecord(ctx, service.db, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (service *DayTrackingService) FindByID(
	ctx context.Context,
	dayRecordID int,
	userID int,
) (*DayRecord, error) {
	record, err := service.dayRecordRepo.FindByID(ctx, dayRecordID, userID)
	if err != nil {
		return nil, err
	}
	if err := service.hydrateRecord(ctx, service.db, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (service *DayTrackingService) FindDateRange(
	ctx context.Context,
	userID int,
	fromDate CalendarDate,
	toDate CalendarDate,
) ([]DayRangeEntry, error) {
	records, err := service.dayRecordRepo.FindByDateRange(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	recordsByDate := make(map[CalendarDate]*DayRecord, len(records))
	for recordIndex := range records {
		record := &records[recordIndex]
		if err := service.hydrateRecord(ctx, service.db, record); err != nil {
			return nil, err
		}
		recordsByDate[record.CalendarDate] = record
	}
	dateCount := int(toDate.Sub(fromDate).Hours()/24) + 1
	entries := make([]DayRangeEntry, 0, dateCount)
	for calendarDate := fromDate; !calendarDate.After(toDate); calendarDate = calendarDate.AddDate(0, 0, 1) {
		entries = append(entries, DayRangeEntry{
			CalendarDate: calendarDate,
			Record:       recordsByDate[calendarDate],
		})
	}
	return entries, nil
}

func (service *DayTrackingService) CreateDayRecord(
	ctx context.Context,
	userID int,
	calendarDate CalendarDate,
	offsetMinutes int,
) (*DayRecord, error) {
	transaction, err := service.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)
	if _, err := service.dayRecordRepo.FindIDForDate(ctx, transaction, userID, calendarDate); err == nil {
		return nil, fmt.Errorf("%w for date %s", ErrDayRecordAlreadyExists, calendarDate.Format(DateFormat))
	} else if !errors.Is(err, ErrDayRecordNotFound) {
		return nil, err
	}
	templateID, snapshotID, err := service.resolveTemplateSnapshotTx(ctx, transaction, userID, calendarDate, nil)
	if err != nil {
		return nil, err
	}
	record, err := service.dayRecordRepo.Create(
		ctx,
		transaction,
		userID,
		calendarDate,
		offsetMinutes,
		templateID,
		snapshotID,
	)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	if err := service.hydrateRecord(ctx, service.db, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (service *DayTrackingService) UpdateTemplateByDate(
	ctx context.Context,
	userID int,
	calendarDate CalendarDate,
	requestedTemplateID *int,
) (*DayRecord, error) {
	transaction, err := service.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)
	if requestedTemplateID != nil {
		exists, err := service.dayTemplateRepo.ExistsForUser(
			ctx,
			transaction,
			*requestedTemplateID,
			userID,
		)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrDayTemplateNotFound
		}
	}
	templateID, snapshotID, err := service.resolveTemplateSnapshotTx(
		ctx,
		transaction,
		userID,
		calendarDate,
		requestedTemplateID,
	)
	if err != nil {
		return nil, err
	}
	recordID, err := service.dayRecordRepo.UpdateTemplate(
		ctx,
		transaction,
		userID,
		calendarDate,
		templateID,
		snapshotID,
	)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	record, err := service.dayRecordRepo.FindByID(ctx, recordID, userID)
	if err != nil {
		return nil, err
	}
	if err := service.hydrateRecord(ctx, service.db, record); err != nil {
		return nil, err
	}
	return record, nil
}

// UpdateTimezoneOffset sets the offset clients use to present the day's UTC
// instants. Stored events and blocks are never touched.
func (service *DayTrackingService) UpdateTimezoneOffset(
	ctx context.Context,
	userID int,
	calendarDate CalendarDate,
	offsetMinutes int,
) (*DayRecord, error) {
	recordID, err := service.dayRecordRepo.UpdateTimezoneOffset(ctx, userID, calendarDate, offsetMinutes)
	if err != nil {
		return nil, err
	}
	return service.FindByID(ctx, recordID, userID)
}

// FindOrCreateDayRecordTx is used by bootstrap; an existing record keeps its offset.
func (service *DayTrackingService) FindOrCreateDayRecordTx(
	ctx context.Context,
	transaction pgx.Tx,
	userID int,
	calendarDate CalendarDate,
	offsetMinutes int,
) (int, error) {
	templateID, snapshotID, err := service.resolveTemplateSnapshotTx(ctx, transaction, userID, calendarDate, nil)
	if err != nil {
		return 0, err
	}
	return service.dayRecordRepo.FindOrCreate(
		ctx,
		transaction,
		userID,
		calendarDate,
		offsetMinutes,
		templateID,
		snapshotID,
	)
}

func (service *DayTrackingService) resolveTemplateSnapshotTx(
	ctx context.Context,
	transaction pgx.Tx,
	userID int,
	calendarDate CalendarDate,
	requestedTemplateID *int,
) (*int, *int, error) {
	templateID := requestedTemplateID
	if templateID == nil {
		var err error
		templateID, err = service.scheduleRepo.ResolveTemplateForDate(
			ctx,
			transaction,
			userID,
			calendarDate,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	if templateID == nil {
		return nil, nil, nil
	}
	snapshotID, err := service.dayTemplateRepo.FindLatestSnapshotID(ctx, transaction, *templateID)
	if err != nil {
		return nil, nil, err
	}
	return templateID, snapshotID, nil
}

func (service *DayTrackingService) hydrateRecord(ctx context.Context, db dbQuerier, record *DayRecord) error {
	record.SnapshotBlocks = make([]SnapshotBlock, 0)
	record.ActualBlocks = make([]ActualBlock, 0)
	record.Events = make([]DayEvent, 0)
	if record.SnapshotID != nil {
		blocks, err := service.dayTemplateRepo.FindSnapshotBlocks(ctx, db, *record.SnapshotID)
		if err != nil {
			return err
		}
		record.SnapshotBlocks = blocks
		record.SnapshottedAt, err = service.dayTemplateRepo.FindSnapshotTime(ctx, db, *record.SnapshotID)
		if err != nil {
			return err
		}
	}
	actualBlocks, err := service.actualBlockRepo.FindByRecordID(ctx, db, record.ID)
	if err != nil {
		return err
	}
	record.ActualBlocks = actualBlocks
	events, err := service.dayEventRepo.FindByRecordID(ctx, db, record.ID)
	if err != nil {
		return err
	}
	record.Events = events
	return nil
}

// ReplaceActualBlocks stores a manually reviewed block list and stops event
// ingestion from rebuilding the day's blocks.
func (service *DayTrackingService) ReplaceActualBlocks(
	ctx context.Context,
	dayRecordID int,
	userID int,
	inputs []ActualBlockInput,
) ([]ActualBlock, error) {
	transaction, err := service.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	if err := service.dayRecordRepo.EnsureOwned(ctx, transaction, dayRecordID, userID); err != nil {
		return nil, err
	}
	blocks, err := service.actualBlockRepo.ReplaceTx(ctx, transaction, dayRecordID, inputs)
	if err != nil {
		return nil, err
	}
	if err := service.dayRecordRepo.MarkBlocksEdited(ctx, transaction, dayRecordID, userID); err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	return blocks, nil
}

// rebuildActualBlocksTx replaces the day's blocks with the ones derived from all events.
func (service *DayTrackingService) rebuildActualBlocksTx(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
) error {
	events, err := service.dayEventRepo.FindByRecordID(ctx, transaction, dayRecordID)
	if err != nil {
		return err
	}
	blocks, err := computeDayTimeline(events)
	if err != nil {
		return err
	}
	_, err = service.actualBlockRepo.ReplaceTx(ctx, transaction, dayRecordID, blocks)
	return err
}

func eventLookupIDs(inputs []DayEventInput) []string {
	lookupIDs := make([]string, 0, len(inputs))
	seenIDs := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if _, alreadySeen := seenIDs[input.ClientEventID]; !alreadySeen {
			seenIDs[input.ClientEventID] = struct{}{}
			lookupIDs = append(lookupIDs, input.ClientEventID)
		}
		if input.EventType != "amendment" {
			continue
		}
		if _, alreadySeen := seenIDs[input.TargetClientEventID]; alreadySeen {
			continue
		}
		seenIDs[input.TargetClientEventID] = struct{}{}
		lookupIDs = append(lookupIDs, input.TargetClientEventID)
	}
	return lookupIDs
}

func prepareNewDateEventInputs(
	inputs []DayEventInput,
	existingEventIDs map[string]struct{},
) ([]DayEventInput, []string, error) {
	batchEventIDs := make(map[string]struct{}, len(inputs))
	processedClientEventIDs := make(map[string]struct{}, len(existingEventIDs)+len(inputs))
	for clientEventID := range existingEventIDs {
		processedClientEventIDs[clientEventID] = struct{}{}
	}
	for _, input := range inputs {
		batchEventIDs[input.ClientEventID] = struct{}{}
	}

	newInputs := make([]DayEventInput, 0, len(inputs))
	duplicateEventIDs := make([]string, 0)
	for _, input := range inputs {
		if _, alreadyProcessed := processedClientEventIDs[input.ClientEventID]; alreadyProcessed {
			duplicateEventIDs = append(duplicateEventIDs, input.ClientEventID)
			continue
		}
		if input.EventType == "amendment" {
			_, targetInBatch := batchEventIDs[input.TargetClientEventID]
			_, targetExists := existingEventIDs[input.TargetClientEventID]
			if !targetInBatch && !targetExists {
				return nil, nil, ErrAmendmentTargetNotFound
			}
		}
		newInputs = append(newInputs, input)
		processedClientEventIDs[input.ClientEventID] = struct{}{}
	}
	return newInputs, duplicateEventIDs, nil
}
