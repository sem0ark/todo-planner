package main

import (
	"context"
	"errors"
)

var (
	ErrInvalidDayDateRange = errors.New("invalid date range")
	ErrDeviceIDRequired    = errors.New("device_id is required")
)

type DayService struct {
	dayRecordRepository *DayRecordRepository
	categoryRepository  *CategoryRepository
}

type DayRangeEntry struct {
	CalendarDate string
	Record       *DayRecord
}

func NewDayService(dayRecordRepository *DayRecordRepository, categoryRepository *CategoryRepository) *DayService {
	return &DayService{
		dayRecordRepository: dayRecordRepository,
		categoryRepository:  categoryRepository,
	}
}

func (service *DayService) GetDays(contextValue context.Context, userID int, fromDate, toDate string) ([]DayRangeEntry, error) {
	fromTime, fromError := parseCalendarDate(fromDate)
	toTime, toError := parseCalendarDate(toDate)
	if fromError != nil || toError != nil || toTime.Before(fromTime) {
		return nil, ErrInvalidDayDateRange
	}
	records, err := service.dayRecordRepository.FindByDateRange(contextValue, userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	recordsByDate := make(map[string]*DayRecord, len(records))
	for recordIndex := range records {
		record := &records[recordIndex]
		recordsByDate[record.CalendarDate] = record
	}

	dateCount := int(toTime.Sub(fromTime).Hours()/24) + 1
	entries := make([]DayRangeEntry, 0, dateCount)
	for currentDate := fromTime; !currentDate.After(toTime); currentDate = currentDate.AddDate(0, 0, 1) {
		calendarDate := currentDate.Format(DateFormat)
		entries = append(entries, DayRangeEntry{CalendarDate: calendarDate, Record: recordsByDate[calendarDate]})
	}
	return entries, nil
}

func (service *DayService) GetDay(contextValue context.Context, userID int, calendarDate string) (*DayRecord, error) {
	return service.dayRecordRepository.FindByDate(contextValue, userID, calendarDate)
}

func (service *DayService) CreateDay(contextValue context.Context, userID int, calendarDate string) (*DayRecord, error) {
	return service.dayRecordRepository.Create(contextValue, userID, calendarDate)
}

func (service *DayService) AppendEvents(contextValue context.Context, userID int, calendarDate string, input DayEventsInput) (*DateEventResult, error) {
	if input.DeviceID <= 0 {
		return nil, ErrDeviceIDRequired
	}
	if err := validateDateEvents(input.Events); err != nil {
		return nil, err
	}
	categoryIDs := make([]int, 0, len(input.Events))
	for _, event := range input.Events {
		if event.CategoryID != nil {
			categoryIDs = append(categoryIDs, *event.CategoryID)
		}
	}
	if err := service.validateCategoryIDs(contextValue, userID, categoryIDs); err != nil {
		return nil, err
	}
	return service.dayRecordRepository.CreateEventsByDate(contextValue, userID, calendarDate, input.DeviceID, input.Events)
}

func (service *DayService) ReplaceBlocks(contextValue context.Context, userID int, calendarDate string, blocks []ActualBlockInput) (*DayRecord, error) {
	if err := validateActualBlocks(blocks); err != nil {
		return nil, err
	}
	categoryIDs := make([]int, 0, len(blocks))
	for _, block := range blocks {
		if block.CategoryID != nil {
			categoryIDs = append(categoryIDs, *block.CategoryID)
		}
	}
	if err := service.validateCategoryIDs(contextValue, userID, categoryIDs); err != nil {
		return nil, err
	}
	record, err := service.dayRecordRepository.FindByDate(contextValue, userID, calendarDate)
	if err != nil {
		return nil, err
	}
	if _, err = service.dayRecordRepository.ReplaceActualBlocks(contextValue, record.ID, userID, blocks); err != nil {
		return nil, err
	}
	return service.dayRecordRepository.FindByDate(contextValue, userID, calendarDate)
}

func (service *DayService) UpdateTemplate(contextValue context.Context, userID int, calendarDate string, templateID *int) (*DayRecord, error) {
	return service.dayRecordRepository.UpdateTemplateByDate(contextValue, userID, calendarDate, templateID)
}

func (service *DayService) validateCategoryIDs(contextValue context.Context, userID int, categoryIDs []int) error {
	validatedCategoryIDs := make(map[int]struct{}, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		if _, alreadyValidated := validatedCategoryIDs[categoryID]; alreadyValidated {
			continue
		}
		validatedCategoryIDs[categoryID] = struct{}{}
		category, err := service.categoryRepository.FindByID(contextValue, categoryID, userID)
		if err != nil || category.IsDeleted {
			return ErrUnknownCategoryID
		}
	}
	return nil
}
