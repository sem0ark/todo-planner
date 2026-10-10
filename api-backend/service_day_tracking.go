package main

import (
	"context"
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

// CreateEvents appends events and refreshes the record's timeline atomically.
func (service *DayTrackingService) CreateEvents(
	ctx context.Context,
	dayRecordID int,
	userID int,
	inputs []DayEventInput,
) ([]DayEvent, []ActualBlock, error) {
	transaction, err := service.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer transaction.Rollback(ctx)

	if err := service.dayRecordRepo.EnsureOwnedTx(ctx, transaction, dayRecordID, userID); err != nil {
		return nil, nil, err
	}

	createdEvents, err := service.dayEventRepo.InsertManualTx(ctx, transaction, dayRecordID, inputs)
	if err != nil {
		return nil, nil, err
	}

	actualBlocks, _, err := service.recomputeActualBlocks(ctx, transaction, dayRecordID)
	if err != nil {
		return nil, nil, err
	}
	if err := service.dayRecordRepo.TouchTx(ctx, transaction, dayRecordID, userID); err != nil {
		return nil, nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return createdEvents, actualBlocks, nil
}

// CreateEventsByDate creates the date record when needed and persists a retry-safe batch.
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

	deviceUserID, err := service.dayEventRepo.FindDeviceUserIDTx(ctx, transaction, deviceID)
	if err != nil {
		return nil, err
	}
	if deviceUserID != userID {
		return nil, ErrDeviceNotFound
	}
	existingDayRecordID, timezoneOffsetLocked, err := service.dayRecordRepo.FindForDateTx(
		ctx,
		transaction,
		userID,
		calendarDate,
	)
	if err != nil {
		return nil, err
	}

	dayRecordID, err := service.findOrCreateDayRecordTx(
		ctx,
		transaction,
		userID,
		calendarDate,
		existingDayRecordID,
	)
	if err != nil {
		return nil, err
	}
	if !timezoneOffsetLocked {
		offsetMinutes, foundOffset, err := firstTimezoneOffset(inputs)
		if err != nil {
			return nil, err
		}
		if foundOffset {
			if err := service.dayRecordRepo.LockTimezoneOffsetTx(
				ctx,
				transaction,
				dayRecordID,
				userID,
				offsetMinutes,
			); err != nil {
				return nil, err
			}
		}
	}

	result := &DateEventResult{
		AcceptedEvents:    make([]DayEvent, 0, len(inputs)),
		DuplicateEventIDs: make([]string, 0),
	}
	existingLocalTimestampsByID, err := service.dayEventRepo.FindExistingLocalTimestampsTx(
		ctx,
		transaction,
		dayRecordID,
		eventLookupIDs(inputs),
	)
	if err != nil {
		return nil, err
	}
	newInputs, duplicateEventIDs, err := prepareNewDateEventInputs(inputs, existingLocalTimestampsByID)
	if err != nil {
		return nil, err
	}
	result.DuplicateEventIDs = duplicateEventIDs
	result.AcceptedEvents, duplicateEventIDs, err = service.dayEventRepo.InsertTx(
		ctx,
		transaction,
		dayRecordID,
		deviceID,
		newInputs,
	)
	if err != nil {
		return nil, err
	}
	result.DuplicateEventIDs = append(result.DuplicateEventIDs, duplicateEventIDs...)

	actualBlocks, events, err := service.recomputeActualBlocks(ctx, transaction, dayRecordID)
	if err != nil {
		return nil, err
	}
	result.Record, err = service.dayRecordRepo.TouchAndFindByIDTx(
		ctx,
		transaction,
		dayRecordID,
		userID,
	)
	if err != nil {
		return nil, err
	}
	result.Record.ActualBlocks = actualBlocks
	result.Record.Events = events
	if result.Record.SnapshotID != nil {
		result.Record.SnapshotBlocks, err = service.dayTemplateRepo.FindSnapshotBlocksTx(
			ctx,
			transaction,
			*result.Record.SnapshotID,
		)
		if err != nil {
			return nil, err
		}
		result.Record.SnapshottedAt, err = service.dayTemplateRepo.FindSnapshotTimeTx(
			ctx,
			transaction,
			*result.Record.SnapshotID,
		)
		if err != nil {
			return nil, err
		}
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
	if err := service.hydrateRecord(ctx, record); err != nil {
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
	if err := service.hydrateRecord(ctx, record); err != nil {
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
		if err := service.hydrateRecord(ctx, record); err != nil {
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
) (*DayRecord, error) {
	transaction, err := service.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)
	if existingRecordID, _, err := service.dayRecordRepo.FindForDateTx(
		ctx,
		transaction,
		userID,
		calendarDate,
	); err != nil {
		return nil, err
	} else if existingRecordID != 0 {
		return nil, fmt.Errorf("%w for date %s", ErrDayRecordAlreadyExists, calendarDate.Format(DateFormat))
	}
	templateID, snapshotID, err := service.resolveTemplateSnapshotTx(ctx, transaction, userID, calendarDate, nil)
	if err != nil {
		return nil, err
	}
	record, err := service.dayRecordRepo.CreateTx(
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
	if err := service.hydrateRecord(ctx, record); err != nil {
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
		exists, err := service.dayTemplateRepo.ExistsForUserTx(
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
	recordID, err := service.dayRecordRepo.UpdateTemplateTx(
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
	if err := service.hydrateRecord(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (service *DayTrackingService) FindOrCreateDayRecordTx(
	ctx context.Context,
	transaction pgx.Tx,
	userID int,
	calendarDate CalendarDate,
) (int, error) {
	existingRecordID, _, err := service.dayRecordRepo.FindForDateTx(
		ctx,
		transaction,
		userID,
		calendarDate,
	)
	if err != nil {
		return 0, err
	}
	return service.findOrCreateDayRecordTx(
		ctx,
		transaction,
		userID,
		calendarDate,
		existingRecordID,
	)
}

func (service *DayTrackingService) findOrCreateDayRecordTx(
	ctx context.Context,
	transaction pgx.Tx,
	userID int,
	calendarDate CalendarDate,
	existingRecordID int,
) (int, error) {
	if existingRecordID != 0 {
		return existingRecordID, nil
	}
	templateID, snapshotID, err := service.resolveTemplateSnapshotTx(
		ctx,
		transaction,
		userID,
		calendarDate,
		nil,
	)
	if err != nil {
		return 0, err
	}
	return service.dayRecordRepo.FindOrCreateTx(
		ctx,
		transaction,
		userID,
		calendarDate,
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
		templateID, err = service.scheduleRepo.ResolveTemplateForDateTx(
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
	snapshotID, err := service.dayTemplateRepo.FindLatestSnapshotIDTx(ctx, transaction, *templateID)
	if err != nil {
		return nil, nil, err
	}
	return templateID, snapshotID, nil
}

func (service *DayTrackingService) hydrateRecord(ctx context.Context, record *DayRecord) error {
	record.SnapshotBlocks = make([]SnapshotBlock, 0)
	record.ActualBlocks = make([]ActualBlock, 0)
	record.Events = make([]DayEvent, 0)
	if record.SnapshotID != nil {
		blocks, err := service.dayTemplateRepo.FindSnapshotBlocks(ctx, *record.SnapshotID)
		if err != nil {
			return err
		}
		record.SnapshotBlocks = blocks
		record.SnapshottedAt, err = service.dayTemplateRepo.FindSnapshotTime(ctx, *record.SnapshotID)
		if err != nil {
			return err
		}
	}
	actualBlocks, err := service.actualBlockRepo.FindByRecordID(ctx, record.ID)
	if err != nil {
		return err
	}
	record.ActualBlocks = actualBlocks
	events, err := service.dayEventRepo.FindByRecordID(ctx, record.ID)
	if err != nil {
		return err
	}
	record.Events = events
	return nil
}

// ReplaceActualBlocks replaces manually tracked blocks and locks the first client timezone.
func (service *DayTrackingService) ReplaceActualBlocks(
	ctx context.Context,
	dayRecordID int,
	userID int,
	inputs []ActualBlockInput,
	clientOffsetMinutes *int,
) ([]ActualBlock, error) {
	transaction, err := service.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback(ctx)

	if err := service.dayRecordRepo.EnsureOwnedTx(ctx, transaction, dayRecordID, userID); err != nil {
		return nil, err
	}
	if clientOffsetMinutes != nil {
		if err := service.dayRecordRepo.LockTimezoneOffsetTx(
			ctx,
			transaction,
			dayRecordID,
			userID,
			*clientOffsetMinutes,
		); err != nil {
			return nil, err
		}
	}
	blocks, err := service.actualBlockRepo.ReplaceTx(ctx, transaction, dayRecordID, inputs)
	if err != nil {
		return nil, err
	}
	if err := service.dayRecordRepo.TouchTx(ctx, transaction, dayRecordID, userID); err != nil {
		return nil, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	return addUntrackedGaps(blocks), nil
}

func (service *DayTrackingService) recomputeActualBlocks(
	ctx context.Context,
	transaction pgx.Tx,
	dayRecordID int,
) ([]ActualBlock, []DayEvent, error) {
	events, err := service.dayEventRepo.FindByRecordIDTx(ctx, transaction, dayRecordID)
	if err != nil {
		return nil, nil, err
	}
	calendarDate, err := service.dayRecordRepo.FindCalendarDateTx(ctx, transaction, dayRecordID)
	if err != nil {
		return nil, nil, err
	}
	existingBlocks, err := service.actualBlockRepo.FindByRecordIDTx(ctx, transaction, dayRecordID)
	if err != nil {
		return nil, nil, err
	}
	computedBlocks, err := computeIncrementalTimeline(events, existingBlocks, calendarDate)
	if err != nil {
		return nil, nil, err
	}
	insertedBlocks, err := service.actualBlockRepo.InsertComputedTx(ctx, transaction, dayRecordID, computedBlocks)
	if err != nil {
		return nil, nil, err
	}
	return append(existingBlocks, insertedBlocks...), events, nil
}

func firstTimezoneOffset(inputs []DayEventInput) (int, bool, error) {
	for _, event := range inputs {
		if event.EventType != "transition" && event.EventType != "confirmation" && event.EventType != "note" {
			continue
		}
		offsetMinutes, err := extractOffsetMinutes(event.OccurredAtLocal)
		if err != nil {
			return 0, false, ErrInvalidLocalTimestamp
		}
		return offsetMinutes, true, nil
	}
	return 0, false, nil
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
	existingLocalTimestampsByID map[string]string,
) ([]DayEventInput, []string, error) {
	batchEventIDs := make(map[string]struct{}, len(inputs))
	localTimestampsByID := make(map[string]string, len(inputs))
	processedClientEventIDs := make(map[string]struct{}, len(existingLocalTimestampsByID)+len(inputs))
	for clientEventID := range existingLocalTimestampsByID {
		processedClientEventIDs[clientEventID] = struct{}{}
	}
	for _, input := range inputs {
		batchEventIDs[input.ClientEventID] = struct{}{}
		localTimestampsByID[input.ClientEventID] = input.OccurredAtLocal
	}

	newInputs := make([]DayEventInput, 0, len(inputs))
	duplicateEventIDs := make([]string, 0)
	for _, input := range inputs {
		if _, alreadyProcessed := processedClientEventIDs[input.ClientEventID]; alreadyProcessed {
			duplicateEventIDs = append(duplicateEventIDs, input.ClientEventID)
			continue
		}
		if err := validateDateEventAmendment(
			input,
			batchEventIDs,
			localTimestampsByID,
			existingLocalTimestampsByID,
		); err != nil {
			return nil, nil, err
		}
		newInputs = append(newInputs, input)
		processedClientEventIDs[input.ClientEventID] = struct{}{}
	}
	return newInputs, duplicateEventIDs, nil
}

func validateDateEventAmendment(
	input DayEventInput,
	batchEventIDs map[string]struct{},
	localTimestampsByID map[string]string,
	existingLocalTimestampsByID map[string]string,
) error {
	if input.EventType != "amendment" {
		return nil
	}
	_, targetInBatch := batchEventIDs[input.TargetClientEventID]
	targetLocal := localTimestampsByID[input.TargetClientEventID]
	if !targetInBatch {
		var targetExists bool
		targetLocal, targetExists = existingLocalTimestampsByID[input.TargetClientEventID]
		if !targetExists {
			return ErrAmendmentTargetNotFound
		}
	}
	if input.CorrectedAtLocal == nil {
		return ErrIncompleteAmendment
	}
	return validateAmendmentOffset(targetLocal, *input.CorrectedAtLocal)
}
