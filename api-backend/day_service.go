package main

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidDayDateRange = errors.New("invalid date range")
	ErrDeviceIDRequired    = errors.New("device_id is required")
)

type DayService struct {
	dayRecordRepository *DayRecordRepository
	categoryRepository  *CategoryRepository
}

func NewDayService(dayRecordRepository *DayRecordRepository, categoryRepository *CategoryRepository) *DayService {
	return &DayService{
		dayRecordRepository: dayRecordRepository,
		categoryRepository:  categoryRepository,
	}
}

func (service *DayService) GetDays(contextValue context.Context, userID int, fromDate, toDate string) ([]DayRecord, error) {
	fromTime, fromError := time.Parse("2006-01-02", fromDate)
	toTime, toError := time.Parse("2006-01-02", toDate)
	if fromError != nil || toError != nil || toTime.Before(fromTime) {
		return nil, ErrInvalidDayDateRange
	}
	return service.dayRecordRepository.FindByDateRange(contextValue, userID, fromDate, toDate)
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
	if err := service.validateCategoryIDs(contextValue, userID, input.Events); err != nil {
		return nil, err
	}
	return service.dayRecordRepository.CreateEventsByDate(contextValue, userID, calendarDate, input.DeviceID, input.Events)
}

func (service *DayService) ReplaceBlocks(contextValue context.Context, userID int, calendarDate string, blocks []ActualBlockInput) (*DayRecord, error) {
	if err := validateActualBlocks(blocks); err != nil {
		return nil, err
	}
	if err := service.validateBlockCategoryIDs(contextValue, userID, blocks); err != nil {
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

func (service *DayService) validateCategoryIDs(contextValue context.Context, userID int, events []DayEventInput) error {
	for _, event := range events {
		if event.CategoryID == nil {
			continue
		}
		category, err := service.categoryRepository.FindByID(contextValue, *event.CategoryID, userID)
		if err != nil || category.IsDeleted {
			return ErrUnknownCategoryID
		}
	}
	return nil
}

func (service *DayService) validateBlockCategoryIDs(contextValue context.Context, userID int, blocks []ActualBlockInput) error {
	for _, block := range blocks {
		if block.CategoryID == nil {
			continue
		}
		category, err := service.categoryRepository.FindByID(contextValue, *block.CategoryID, userID)
		if err != nil || category.IsDeleted {
			return ErrUnknownCategoryID
		}
	}
	return nil
}
