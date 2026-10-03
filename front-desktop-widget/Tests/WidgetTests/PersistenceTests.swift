import Foundation

@MainActor
final class PersistenceTests: WidgetTestCase {

  func test_bootstrapCache_roundTripsAndRejectsOtherDates() throws {
    let (cache, directory) = try makeTemporaryBootstrapCache()
    defer { try? FileManager.default.removeItem(at: directory) }
    let response = makeBootstrapResponse()

    try cache.save(response: response, calendarDate: Fixtures.today)

    let loadedResponse = try cache.load(calendarDate: Fixtures.today)?.response
    try assertEqual(loadedResponse?.dayRecords[0].calendarDate, Fixtures.today)
    try assertEqual(loadedResponse?.categories.map(\.id), [Fixtures.categoryA.id])
    try assertEqual(
      try cache.load(calendarDate: Fixtures.today)?.response.dayRecords[0].calendarDate,
      Fixtures.today)
    try assert(cache.load(calendarDate: "2099-01-01") == nil, "Cache must be date scoped")

    try cache.removeAll()
    try assert(cache.load(calendarDate: Fixtures.today) == nil, "Cache should be removable")
  }

  func test_bootstrapCache_rejectsMismatchedResponseDate() throws {
    let (cache, directory) = try makeTemporaryBootstrapCache()
    defer { try? FileManager.default.removeItem(at: directory) }
    let response = makeBootstrapResponse(calendarDate: "2099-01-01")

    try assert(
      (try? cache.save(response: response, calendarDate: Fixtures.today)) == nil,
      "Cache must reject a response for another date")
  }

  func test_bootstrapCache_returnsNilWhenTodayIsMissing() throws {
    let (cache, directory) = try makeTemporaryBootstrapCache()
    defer { try? FileManager.default.removeItem(at: directory) }

    try cache.save(
      response: makeBootstrapResponse(calendarDate: "2026-09-19"),
      calendarDate: "2026-09-19")
    try cache.save(
      response: makeBootstrapResponse(calendarDate: "2026-09-26"),
      calendarDate: "2026-09-26")

    let cachedResponse = try cache.load(calendarDate: Fixtures.today)

    try assert(cachedResponse == nil, "Cache must not use another calendar date")
  }

  func test_localEventStoreBackupSurvivesQueueRemoval() throws {
    let (eventStore, directory) = try makeTemporaryEventStore()
    defer { try? FileManager.default.removeItem(at: directory) }
    let event = Fixtures.event(id: "backup-event", occurredAt: Fixtures.now)

    try eventStore.append(calendarDate: Fixtures.today, event: event)
    try eventStore.remove(clientEventIds: [event.clientEventId])

    try assert(eventStore.pendingEvents().isEmpty, "Queue should be empty after removal")
    try assertEqual(eventStore.backupEvents().map(\.event.clientEventId), [event.clientEventId])
  }

  func test_localEventStoreMigratesLegacyJSONQueueToJSONLines() throws {
    let (eventStore, directory) = try makeTemporaryEventStore()
    defer { try? FileManager.default.removeItem(at: directory) }
    let event = Fixtures.event(id: "legacy-event", occurredAt: Fixtures.now)
    let entry = PendingDayEvent(calendarDate: Fixtures.today, event: event)
    let legacyData = try JSONEncoder.widgetEncoder.encode([entry])
    try legacyData.write(to: directory.appendingPathComponent("pending-events.json"))

    try assertEqual(eventStore.pendingEvents().map(\.event.clientEventId), [event.clientEventId])
    try eventStore.append(
      calendarDate: Fixtures.today,
      event: Fixtures.event(id: "new-event", occurredAt: Fixtures.now.addingTimeInterval(60)))

    let migratedData = try Data(contentsOf: directory.appendingPathComponent("pending-events.jsonl"))
    try assert(migratedData.first == 0x7B,
      "Migrated queue should use JSON Lines rather than a JSON array")
    try assertEqual(eventStore.pendingEvents().count, 2)
  }

  func test_localEventStoreBackfillsMissingLocalTimestamps() throws {
    let (eventStore, directory) = try makeTemporaryEventStore()
    defer { try? FileManager.default.removeItem(at: directory) }
    let queueJSON = """
    {"calendarDate":"2026-09-27","event":{"client_event_id":"legacy-event","event_type":"amendment","category_id":null,"occurred_at":"2026-09-27T10:00:00Z","corrected_at":"2026-09-27T09:45:00Z","target_client_event_id":"target-event"}}
    """.data(using: .utf8)!
    try queueJSON.write(to: directory.appendingPathComponent("pending-events.jsonl"))

    let event = try eventStore.pendingEvents()[0].event
    guard let correctedAt = event.correctedAt else {
      throw AssertionError.failed("Legacy corrected_at should decode")
    }

    try assertEqual(event.occurredAtLocal, TimeFormats.localTimestamp(for: event.occurredAt))
    try assertEqual(event.correctedAtLocal, TimeFormats.localTimestamp(for: correctedAt))
  }

  func test_appendFailurePreservesQueueAndBackup() throws {
    let (eventStore, directory) = try makeTemporaryEventStore()
    defer { try? FileManager.default.removeItem(at: directory) }
    let existingEvent = Fixtures.event(id: "existing", occurredAt: Fixtures.now)
    try eventStore.append(calendarDate: Fixtures.today, event: existingEvent)
    let failingStore = LocalEventStore(
      storageDirectory: directory,
      appendFile: { _, _ in throw StorageError.databaseError("append stopped") })

    let failedEvent = Fixtures.event(id: "failed", occurredAt: Fixtures.now.addingTimeInterval(60))
    try assert((try? failingStore.append(calendarDate: Fixtures.today, event: failedEvent)) == nil,
      "Append failures should be reported")
    try assertEqual(eventStore.pendingEvents().map(\.event.clientEventId), [existingEvent.clientEventId])
    try assertEqual(eventStore.backupEvents().map(\.event.clientEventId), [existingEvent.clientEventId])
  }

  func test_atomicQueueRewriteFailurePreservesContents() throws {
    let (eventStore, directory) = try makeTemporaryEventStore()
    defer { try? FileManager.default.removeItem(at: directory) }
    let firstEvent = Fixtures.event(id: "first", occurredAt: Fixtures.now)
    let secondEvent = Fixtures.event(id: "second", occurredAt: Fixtures.now.addingTimeInterval(60))
    try eventStore.append(calendarDate: Fixtures.today, event: firstEvent)
    try eventStore.append(calendarDate: Fixtures.today, event: secondEvent)
    let failingStore = LocalEventStore(
      storageDirectory: directory,
      atomicWriteFile: { _, _ in throw StorageError.databaseError("atomic write stopped") })

    try assert((try? failingStore.remove(clientEventIds: [firstEvent.clientEventId])) == nil,
      "Queue rewrite failures should be reported")
    try assertEqual(eventStore.pendingEvents().map(\.event.clientEventId),
      [firstEvent.clientEventId, secondEvent.clientEventId])
  }

  func test_clearLocalDataRemovesEventsAndBootstrapCache() throws {
    let (eventStore, eventDirectory) = try makeTemporaryEventStore()
    let (cache, cacheDirectory) = try makeTemporaryBootstrapCache()
    defer {
      try? FileManager.default.removeItem(at: eventDirectory)
      try? FileManager.default.removeItem(at: cacheDirectory)
    }
    let event = Fixtures.event(id: "logout-event", occurredAt: Fixtures.now)
    try eventStore.append(calendarDate: Fixtures.today, event: event)
    try cache.save(response: makeBootstrapResponse(), calendarDate: Fixtures.today)
    let repository = RemoteTodoPlannerRepository(
      api: SynchronizationAPI(), eventStore: eventStore, bootstrapCache: cache)

    try repository.clearLocalData()

    try assert(eventStore.pendingEvents().isEmpty, "Logout should clear pending events")
    try assert(eventStore.backupEvents().isEmpty, "Logout should clear backup events")
    try assert(cache.load(calendarDate: Fixtures.today) == nil, "Logout should clear bootstrap cache")
  }

  static func testMethods() -> [TestCase] {
    let tests = PersistenceTests()
    return [
      ("test_bootstrapCache_roundTripsAndRejectsOtherDates", { try tests.test_bootstrapCache_roundTripsAndRejectsOtherDates() }),
      ("test_bootstrapCache_rejectsMismatchedResponseDate", { try tests.test_bootstrapCache_rejectsMismatchedResponseDate() }),
      ("test_bootstrapCache_returnsNilWhenTodayIsMissing", { try tests.test_bootstrapCache_returnsNilWhenTodayIsMissing() }),
      ("test_localEventStoreBackupSurvivesQueueRemoval", { try tests.test_localEventStoreBackupSurvivesQueueRemoval() }),
      ("test_localEventStoreMigratesLegacyJSONQueueToJSONLines", { try tests.test_localEventStoreMigratesLegacyJSONQueueToJSONLines() }),
      ("test_localEventStoreBackfillsMissingLocalTimestamps", { try tests.test_localEventStoreBackfillsMissingLocalTimestamps() }),
      ("test_appendFailurePreservesQueueAndBackup", { try tests.test_appendFailurePreservesQueueAndBackup() }),
      ("test_atomicQueueRewriteFailurePreservesContents", { try tests.test_atomicQueueRewriteFailurePreservesContents() }),
      ("test_clearLocalDataRemovesEventsAndBootstrapCache", { try tests.test_clearLocalDataRemovesEventsAndBootstrapCache() }),
    ]
  }
}
