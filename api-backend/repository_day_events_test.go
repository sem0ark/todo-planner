package main

import (
	"context"
	"testing"
)

func TestDayEventRepository_InsertAndFindByRecordID(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "day-event-repository", "password123")
	device, err := NewDeviceRepository(database).Create(context.Background(), user.ID, "desktop")
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	record, err := newDayTrackingServiceForTest(database).CreateDayRecord(
		context.Background(), user.ID, mustCalendarDate("2026-09-06"),
	)
	if err != nil {
		t.Fatalf("failed to create day record: %v", err)
	}
	eventRepository := NewDayEventRepository(database)
	transaction, err := database.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	createdEvents, duplicateEventIDs, err := eventRepository.InsertTx(
		context.Background(),
		transaction,
		record.ID,
		device.ID,
		[]DayEventInput{{
			ClientEventID:   "repository-event",
			EventType:       "note",
			OccurredAt:      parseTime("2026-09-06T09:00:00Z"),
			OccurredAtLocal: "2026-09-06T09:00:00+00:00",
		}},
	)
	if err != nil {
		transaction.Rollback(context.Background())
		t.Fatalf("failed to insert event: %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("failed to commit event: %v", err)
	}
	if len(createdEvents) != 1 || len(duplicateEventIDs) != 0 {
		t.Fatalf("unexpected insert result: events=%d duplicates=%v", len(createdEvents), duplicateEventIDs)
	}
	loadedEvents, err := eventRepository.FindByRecordID(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("failed to load events: %v", err)
	}
	if len(loadedEvents) != 1 || loadedEvents[0].ClientEventID == nil || *loadedEvents[0].ClientEventID != "repository-event" {
		t.Fatalf("unexpected loaded events: %+v", loadedEvents)
	}
}
