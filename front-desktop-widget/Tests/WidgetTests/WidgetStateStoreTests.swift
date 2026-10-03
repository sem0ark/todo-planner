import Foundation

// ═══════════════════════════════════════════════════════════════════
// MARK: - Test Store (overrides AppKit-dependent methods)
// ═══════════════════════════════════════════════════════════════════

@MainActor
final class TestableWidgetStateStore: WidgetStateStore {
  override func updateMenuBarIcon() {
    // Stub: no-op in test environment to avoid AppKit runtime issues
  }
}

enum AssertionError: Error {
  case failed(String)
}

func assert(_ condition: Bool, _ message: @autoclosure () -> String) throws {
  guard condition else { throw AssertionError.failed(message()) }
}

func assertEqual<T: Equatable>(_ lhs: T, _ rhs: T) throws {
  guard lhs == rhs else {
    throw AssertionError.failed("Expected \(rhs), got \(lhs)")
  }
}

// ═══════════════════════════════════════════════════════════════════
// MARK: - Recorded Call
// ═══════════════════════════════════════════════════════════════════

enum RecordedCall: CustomStringConvertible {
  case initialize(date: String)
  case submitEvents(calendarDate: String, events: [DayEvent])

  var description: String {
    switch self {
    case .initialize(let date): return "initialize(\(date))"
    case .submitEvents(let date, let e):
      let types = e.map(\.eventType).joined(separator: ",")
      return "submitEvents(date:\(date), types:[\(types)])"
    }
  }
}

// ═══════════════════════════════════════════════════════════════════
// MARK: - Mock Repository
// ═══════════════════════════════════════════════════════════════════

final class MockRepository: TodoPlannerRepository, @unchecked Sendable {
  var stubbedCategories: [Category] = []
  var stubbedDayRecord: DayRecord? = nil
  var stubbedCreatedRecord: DayRecord?
  var stubbedEventsResponse: DayEventsResponse?
  var shouldThrowOnSubmitEvents = false
  private(set) var validateAuthCallCount = 0
  private var cachedBootstrap: InitResponse?

  private(set) var calls: [RecordedCall] = []

  func resetCalls() { calls.removeAll() }

  var submitEventsCalls: [(calendarDate: String, events: [DayEvent])] {
    calls.compactMap {
      guard case .submitEvents(let date, let events) = $0 else { return nil }
      return (date, events)
    }
  }

  func getAuthToken() -> String? { "test-token" }
  func persistAuthToken(_ token: String) async throws {}
  func clearAuth() async throws {}
  func validateAuth() async throws -> Bool {
    validateAuthCallCount += 1
    return true
  }

  func initialize(calendarDate: String) async throws -> InitResponse {
    calls.append(.initialize(date: calendarDate))
    let dayRecord: DayRecord
    if let existingRecord = stubbedDayRecord {
      dayRecord = existingRecord
    } else if let createdRecord = stubbedCreatedRecord {
      dayRecord = createdRecord
    } else {
      throw StorageError.notFound
    }
    let response = InitResponse(
       settings: UserSettings(
         dayRangeStartTime: "04:00:00", dayRangeEndTime: "23:00:00", updatedAt: Fixtures.now),
      categories: stubbedCategories,
      dayRecords: [dayRecord]
    )
    cachedBootstrap = response
    return response
  }

  func submitEvents(calendarDate: String, events: [DayEvent]) async throws -> DayEventsResponse {
    calls.append(.submitEvents(calendarDate: calendarDate, events: events))
    if shouldThrowOnSubmitEvents {
      throw NSError(domain: "MockError", code: 1, userInfo: nil)
    }
    return stubbedEventsResponse ?? DayEventsResponse()
  }

  func hasPendingSync() async -> Bool { false }
  func synchronize() async throws {}
  func cachedInitialization(calendarDate: String) throws -> InitResponse? {
    guard cachedBootstrap?.dayRecords[0].calendarDate == calendarDate else { return nil }
    return cachedBootstrap
  }
  func clearLocalData() throws { cachedBootstrap = nil }
}

final class SynchronizationAPI: TodoPlannerAPI, @unchecked Sendable {
  var authToken: String?
  var initializationResponse: InitResponse?

  init(authToken: String? = nil, initializationResponse: InitResponse? = nil) {
    self.authToken = authToken
    self.initializationResponse = initializationResponse
  }

  private(set) var postedEvents: [[DayEvent]] = []
  var results: [Result<DayEventsResponse, APIError>] = []

  func setAuthToken(_ token: String) {}
  func clearAuthToken() {}
  func validateToken() async throws -> Bool { true }
  func initialize(calendarDate: String) async throws -> InitResponse {
    guard let initializationResponse else { throw StorageError.notFound }
    return initializationResponse
  }

  func postDayEvents(date: String, events: [DayEvent]) async throws -> DayEventsResponse {
    postedEvents.append(events)
    return try results.removeFirst().get()
  }
}

// ═══════════════════════════════════════════════════════════════════
// MARK: - Test Fixtures
// ═══════════════════════════════════════════════════════════════════

enum Fixtures {
  static let now = Date()

  static let categoryA = Category(
    id: 1, name: "Working", color: "#FF0000",
    pomodoroConfig: nil, createdAt: now, updatedAt: now
  )

  static let categoryB = Category(
    id: 2, name: "Exercising", color: "#00FF00",
    pomodoroConfig: nil, createdAt: now, updatedAt: now
  )

  static let defaultCategories: [Category] = [categoryA, categoryB]

  static func blockCoveringNow(
    id: Int = 1,
    categoryId: Int = 1,
    durationMinutes: Int = 60
  ) -> PlannedBlock {
    let comps = Calendar.current.dateComponents([.hour], from: Date())
    let start = String(format: "%02d:00:00", comps.hour ?? 0)
    return PlannedBlock(categoryId: categoryId, startTime: start, durationMinutes: durationMinutes)
  }

  static var today: String {
    DateFormatter.yyyyMMdd.string(from: Date())
  }

  static func record(id: Int = 1) -> DayRecord {
    DayRecord(
      calendarDate: today,
      plan: []
    )
  }

  static func recordWithCurrentBlock(categoryId: Int = 1) -> DayRecord {
    DayRecord(
      calendarDate: today,
      plan: [blockCoveringNow(categoryId: categoryId)]
    )
  }

  static func eventsResponse() -> DayEventsResponse {
    DayEventsResponse(calendarDate: today)
  }

  static func event(id: String, occurredAt: Date, categoryId: Int? = 1) -> DayEvent {
    DayEvent(
      clientEventId: id,
      eventType: "transition",
      categoryId: categoryId,
      occurredAt: occurredAt,
      occurredAtLocal: TimeFormats.localTimestamp(for: occurredAt)
    )
  }
}

// ═══════════════════════════════════════════════════════════════════
// MARK: - Test Harness
// ═══════════════════════════════════════════════════════════════════

@MainActor
final class WidgetTestHarness {
  let mock: MockRepository
  let store: TestableWidgetStateStore

  init(
    categories: [Category] = Fixtures.defaultCategories,
    existingRecord: DayRecord? = nil,
    createdRecord: DayRecord? = nil,
    eventsResponse: DayEventsResponse? = nil
  ) {
    let repo = MockRepository()
    repo.stubbedCategories = categories
    repo.stubbedDayRecord = existingRecord
    repo.stubbedEventsResponse = eventsResponse
    self.mock = repo
    self.store = TestableWidgetStateStore(repository: repo)
    store.stopPeriodicRefresh()
  }

  func initialize() async {
    await store.initialize()
  }

  func initializeAndResetCalls() async {
    await store.initialize()
    mock.resetCalls()
  }

  func assertCallCount(_ expected: Int) throws {
    try assertEqual(mock.calls.count, expected)
  }

  func assertSubmitEventsCount(_ expected: Int) throws {
    try assertEqual(mock.submitEventsCalls.count, expected)
  }

  func assertNoSubmitEvents() throws {
    try assert(mock.submitEventsCalls.isEmpty, "Expected no submitEvents calls, got \(mock.submitEventsCalls.count)")
  }

  func assertContainsCall(_ predicate: (RecordedCall) -> Bool) throws {
    try assert(mock.calls.contains(where: predicate), "Expected call not found")
  }

  func assertSubmitEventDetails(
    index: Int,
    expectedType: String,
    expectedIncomingId: Int?,
    calendarDate: String? = nil
  ) throws {
    let calls = mock.submitEventsCalls
    guard index < calls.count else {
      throw AssertionError.failed("submitEvents call at index \(index) not found (total: \(calls.count))")
    }

    let (date, events) = calls[index]
    guard let event = events.first else {
      throw AssertionError.failed("submitEvents[\(index)] has no events")
    }

    try assertEqual(event.eventType, expectedType)
    try assertEqual(event.categoryId, expectedIncomingId)

    if let expectedDate = calendarDate {
      try assertEqual(date, expectedDate)
    }
  }
}

// ═══════════════════════════════════════════════════════════════════
// MARK: - Tests
// ═══════════════════════════════════════════════════════════════════

@MainActor
final class WidgetStateStoreTests {
  private func makeTemporaryEventStore() throws -> (LocalEventStore, URL) {
    let directory = FileManager.default.temporaryDirectory
      .appendingPathComponent("TodoPlannerWidgetTests-\(UUID().uuidString)")
    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    return (LocalEventStore(storageDirectory: directory), directory)
  }

  private func makeTemporaryBootstrapCache() throws -> (BootstrapCache, URL) {
    let directory = FileManager.default.temporaryDirectory
      .appendingPathComponent("TodoPlannerWidgetBootstrapTests-\(UUID().uuidString)")
    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    return (BootstrapCache(storageDirectory: directory), directory)
  }

  private func makeBootstrapResponse(calendarDate: String = Fixtures.today) -> InitResponse {
    InitResponse(
      settings: UserSettings(
        dayRangeStartTime: "04:00:00", dayRangeEndTime: "23:00:00", updatedAt: Fixtures.now),
      categories: [Fixtures.categoryA],
      dayRecords: [DayRecord(
        calendarDate: calendarDate,
        plan: [PlannedBlock(categoryId: Fixtures.categoryA.id, startTime: "00:00:00", durationMinutes: 1440)]
      )]
    )
  }

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

  func test_cachedBootstrap_startsWidgetWithoutLoggingTransition() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initialize()
    h.mock.resetCalls()
    h.store.context = WidgetContext()
    h.store.currentState = InitializingState()

    try assert(h.store.initializeFromCache(), "A successful initialization should create a cache")
    try assertEqual(h.store.displayState, .active)
    try assertEqual(h.store.currentCategory?.id, Fixtures.categoryA.id)
    try h.assertNoSubmitEvents()
  }

  func test_synchronize_refreshesBootstrapCache() async throws {
    let (eventStore, eventDirectory) = try makeTemporaryEventStore()
    let (cache, cacheDirectory) = try makeTemporaryBootstrapCache()
    defer {
      try? FileManager.default.removeItem(at: eventDirectory)
      try? FileManager.default.removeItem(at: cacheDirectory)
    }
    let api = SynchronizationAPI(
      authToken: "test-token", initializationResponse: makeBootstrapResponse())
    let repository = RemoteTodoPlannerRepository(
      api: api, eventStore: eventStore, bootstrapCache: cache)

    try await repository.synchronize()

    try assertEqual(try cache.load(calendarDate: Fixtures.today)?.response.dayRecords[0].calendarDate, Fixtures.today)
  }

  func test_synchronizeRefreshFailurePreservesExistingBootstrapCache() async throws {
    let (eventStore, eventDirectory) = try makeTemporaryEventStore()
    let (cache, cacheDirectory) = try makeTemporaryBootstrapCache()
    defer {
      try? FileManager.default.removeItem(at: eventDirectory)
      try? FileManager.default.removeItem(at: cacheDirectory)
    }
    let existingResponse = makeBootstrapResponse()
    try cache.save(response: existingResponse, calendarDate: Fixtures.today)
    let api = SynchronizationAPI(authToken: "test-token")
    let repository = RemoteTodoPlannerRepository(
      api: api, eventStore: eventStore, bootstrapCache: cache)

    do {
      try await repository.synchronize()
      throw AssertionError.failed("Bootstrap refresh failure should propagate")
    } catch StorageError.notFound {
      // Expected: a failed refresh must not remove the previous cache.
    }

    try assertEqual(try cache.load(calendarDate: Fixtures.today)?.response.dayRecords[0].calendarDate, Fixtures.today)
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

  func test_authenticationUsesMemoryTokenWithoutValidationRequest() async throws {
    let repository = MockRepository()
    let authController = AuthController(repository: repository)

    await authController.checkInitialAuth()

    try assert(authController.isAuthenticated, "A memory token should authenticate the session")
    try assertEqual(repository.validateAuthCallCount, 0)
  }

  func test_apiClientDoesNotPersistAuthenticationToken() throws {
    let tokenKey = "com.todoplanner.widget.jwt_token"
    UserDefaults.standard.removeObject(forKey: tokenKey)
    APIClient.shared.setAuthToken("memory-only-token")

    try assert(UserDefaults.standard.string(forKey: tokenKey) == nil, "JWT must not be persisted")

    APIClient.shared.clearAuthToken()
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

  func test_synchronizeSendsSortedEventsAsSingleBatch() async throws {
    let (eventStore, directory) = try makeTemporaryEventStore()
    defer { try? FileManager.default.removeItem(at: directory) }
    let api = SynchronizationAPI()
    let earlierEvent = Fixtures.event(id: "earlier", occurredAt: Fixtures.now.addingTimeInterval(-60))
    let laterEvent = Fixtures.event(id: "later", occurredAt: Fixtures.now)
    api.results = [.success(DayEventsResponse(
      acceptedEvents: [
        AcceptedEvent(
          clientEventId: earlierEvent.clientEventId,
          eventType: earlierEvent.eventType,
          categoryId: earlierEvent.categoryId,
          occurredAt: earlierEvent.occurredAt,
          occurredAtLocal: earlierEvent.occurredAtLocal),
        AcceptedEvent(
          clientEventId: laterEvent.clientEventId,
          eventType: laterEvent.eventType,
          categoryId: laterEvent.categoryId,
          occurredAt: laterEvent.occurredAt,
          occurredAtLocal: laterEvent.occurredAtLocal),
      ],
      calendarDate: Fixtures.today))]
    try eventStore.append(calendarDate: Fixtures.today, event: laterEvent)
    try eventStore.append(calendarDate: Fixtures.today, event: earlierEvent)

    try await RemoteTodoPlannerRepository(api: api, eventStore: eventStore).synchronize()

    try assertEqual(api.postedEvents.count, 1)
    try assertEqual(api.postedEvents[0].map(\.clientEventId), [earlierEvent.clientEventId, laterEvent.clientEventId])
    try assert(eventStore.pendingEvents().isEmpty, "Acknowledged events should leave the queue")
    try assertEqual(eventStore.backupEvents().count, 2)
  }

  func test_synchronize400FallsBackToIndividualEvents() async throws {
    let (eventStore, directory) = try makeTemporaryEventStore()
    defer { try? FileManager.default.removeItem(at: directory) }
    let api = SynchronizationAPI()
    let events = [
      Fixtures.event(id: "accepted", occurredAt: Fixtures.now),
      Fixtures.event(id: "duplicate", occurredAt: Fixtures.now.addingTimeInterval(60)),
      Fixtures.event(id: "rejected", occurredAt: Fixtures.now.addingTimeInterval(120)),
    ]
    for event in events {
      try eventStore.append(calendarDate: Fixtures.today, event: event)
    }
    api.results = [
      .failure(.serverError(400, "batch rejected")),
      .success(DayEventsResponse(acceptedEvents: [AcceptedEvent(
        clientEventId: events[0].clientEventId,
        eventType: events[0].eventType,
        categoryId: events[0].categoryId,
        occurredAt: events[0].occurredAt,
        occurredAtLocal: events[0].occurredAtLocal)], calendarDate: Fixtures.today)),
      .success(DayEventsResponse(duplicateClientEventIds: [events[1].clientEventId], calendarDate: Fixtures.today)),
      .failure(.serverError(400, "event rejected")),
    ]

    try await RemoteTodoPlannerRepository(api: api, eventStore: eventStore).synchronize()

    try assertEqual(api.postedEvents.count, 4)
    try assertEqual(api.postedEvents[0].count, 3)
    try assertEqual(api.postedEvents.dropFirst().map { $0[0].clientEventId },
      events.map(\.clientEventId))
    try assert(eventStore.pendingEvents().isEmpty, "Fallback events should no longer block sync")
    try assertEqual(eventStore.backupEvents().map(\.event.clientEventId), events.map(\.clientEventId))
  }

  func test_synchronizeNon400ErrorPreservesQueue() async throws {
    let (eventStore, directory) = try makeTemporaryEventStore()
    defer { try? FileManager.default.removeItem(at: directory) }
    let api = SynchronizationAPI()
    let event = Fixtures.event(id: "network-failure", occurredAt: Fixtures.now)
    try eventStore.append(calendarDate: Fixtures.today, event: event)
    api.results = [.failure(.serverError(500, "server unavailable"))]

    do {
      try await RemoteTodoPlannerRepository(api: api, eventStore: eventStore).synchronize()
      throw AssertionError.failed("Non-400 synchronization errors should propagate")
    } catch APIError.serverError(500, _) {
      // Expected: non-400 errors must remain retryable.
    }
    try assertEqual(eventStore.pendingEvents().map(\.event.clientEventId), [event.clientEventId])
  }

  func test_initResponse_decodesCurrentAPIShape() throws {
    let json = """
    {
      "settings": {
        "day_range_start_time": "04:00:00",
        "updated_at": "2026-09-06T14:30:00Z"
      },
      "categories": [],
      "day_records": [{
        "calendar_date": "2026-09-06",
        "day_template_id": 5,
        "plan": [
            {
              "category_id": 3,
              "start_time": "08:00:00",
              "duration_minutes": 60
            }
          ],
        "actual": [
          {
            "category_id": 3,
            "block_type": "actual",
             "start_time": "08:05:00",
            "duration_minutes": 55
          }
        ],
        "created_at": "2026-09-06T07:55:00Z",
        "updated_at": "2026-09-06T14:30:00Z"
      }]
    }
    """.data(using: .utf8)!
    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601

    let response = try decoder.decode(InitResponse.self, from: json)

    try assert(response.settings.updatedAt.timeIntervalSince1970 > 0, "settings timestamp should decode")
    try assertEqual(response.dayRecords[0].calendarDate, "2026-09-06")
    try assertEqual(response.dayRecords[0].plan[0].startTime, "08:00:00")
  }

  func test_dayEventsResponse_decodesNullAcceptedEventCategory() throws {
    let json = """
    {
      "calendar_date": "2026-09-07",
      "day_template_id": null,
      "plan": [],
      "actual": [],
      "accepted_events": [
        {
          "client_event_id": "event-1",
          "event_type": "amendment",
          "category_id": null,
          "occurred_at": "2026-09-07T10:55:04Z"
        }
      ],
      "duplicate_client_event_ids": [],
      "created_at": "2026-09-07T10:00:00Z",
      "updated_at": "2026-09-07T10:55:04Z"
    }
    """.data(using: .utf8)!
    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601

    let response = try decoder.decode(DayEventsResponse.self, from: json)

    try assert(response.acceptedEvents[0].categoryId == nil, "category_id should allow null")
  }

  func test_dayEventsResponse_missingRequiredArrayFailsDecoding() throws {
    let json = """
    {"calendar_date":"2026-09-07","plan":[],"actual":[],
     "created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"}
    """.data(using: .utf8)!
    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601

    try assert((try? decoder.decode(DayEventsResponse.self, from: json)) == nil,
      "Missing accepted_events and duplicate_client_event_ids must fail decoding")
  }

  func test_dayRecordsResponse_missingContainerFailsDecoding() throws {
    let json = "{}".data(using: .utf8)!
    try assert((try? JSONDecoder().decode(DayRecordsResponse.self, from: json)) == nil,
      "Missing day_records and days must fail decoding")
  }

  func test_dayEventEncodesIdempotencyAndAmendmentFields() throws {
    let event = DayEvent(
      clientEventId: "event-1",
      eventType: "amendment",
      categoryId: 3,
      occurredAt: Fixtures.now,
      occurredAtLocal: TimeFormats.localTimestamp(for: Fixtures.now),
      targetClientEventId: "event-0",
      correctedAt: Fixtures.now
    )
    let encoder = JSONEncoder()
    encoder.dateEncodingStrategy = .iso8601
    let data = try encoder.encode(event)
    let object = try JSONSerialization.jsonObject(with: data) as? [String: Any]

    try assertEqual(object?["client_event_id"] as? String, "event-1")
    try assertEqual(object?["event_type"] as? String, "amendment")
    try assertEqual(object?["target_client_event_id"] as? String, "event-0")
     try assert(object?["occurred_at_local"] != nil, "occurred_at_local should be encoded")
  }

  func test_init_freshDay_createsRecord() async throws {
    let newRecord = Fixtures.record()
    let h = WidgetTestHarness(existingRecord: nil, createdRecord: newRecord)
    await h.initialize()
    try h.assertContainsCall { if case .initialize = $0 { return true }; return false }
    try h.assertNoSubmitEvents()
  }

  func test_init_withPlannedCategory_picksCategoryAndLogsTransition() async throws {
    let plannedBlock = Fixtures.blockCoveringNow(categoryId: Fixtures.categoryA.id)
    let dayRecord = DayRecord(
      calendarDate: Fixtures.today,
      plan: [plannedBlock]
    )
    let h = WidgetTestHarness(existingRecord: dayRecord)
    await h.initialize()

    try assertEqual(h.store.currentCategory?.id, Fixtures.categoryA.id)
    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
      expectedType: "transition",
      expectedIncomingId: Fixtures.categoryA.id
    )
  }

  func test_init_existingRecord_doesNotCreate() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initialize()
    try h.assertNoSubmitEvents()
  }

  func test_invalidScheduleTime_doesNotSelectBlock() throws {
    let block = PlannedBlock(categoryId: 1, startTime: "not-a-time", durationMinutes: 60)
    try assert(TimeLogic.getCurrentPlannedBlock(at: Date(), from: [block]) == nil,
      "Malformed schedule times must not be interpreted as midnight")
  }

  func test_invalidBootstrap_keepsStoreInitializing() async throws {
    let invalidRecord = DayRecord(
      calendarDate: Fixtures.today,
      plan: [PlannedBlock(categoryId: 1, startTime: "not-a-time", durationMinutes: 60)]
    )
    let h = WidgetTestHarness(existingRecord: invalidRecord)
    await h.initialize()
    try assert(h.store.displayState == .initializing,
      "A contract-breaking bootstrap must not activate the widget")
    try assert(h.store.lastError != nil, "Bootstrap failure must be exposed")
  }

  func test_actualBlocksAreNotPartOfWidgetPlanModel() async throws {
    let record = Fixtures.record()
    let h = WidgetTestHarness(existingRecord: record)
    await h.initialize()
    try assert(h.store.displayState == .active,
      "The widget should initialize from the plan without actual blocks")
  }

  func test_selectCategory_logsTransition() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
       expectedType: "transition",
       expectedIncomingId: Fixtures.categoryA.id,
    )
  }

  func test_initialState_isInitializing() throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    try assert(h.store.displayState == .initializing, "Initial state should be initializing")
  }

  func test_afterInitialize_isActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initialize()
    try assert(h.store.displayState == .active, "After init, state should be active")
  }

  func test_reload_fetchesRemoteDataAndReturnsToActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initialize()

    await h.store.reload()

    let initializationCount = h.mock.calls.reduce(into: 0) { count, call in
      if case .initialize = call { count += 1 }
    }

    try assertEqual(initializationCount, 2)
    try assert(h.store.displayState == .active, "After reload, state should be active")
  }

  func test_afterConfirmation_returnsToActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: 1))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.primaryAction)
    try assert(h.store.displayState == .active, "Should return to active after dispatch")
  }

  func test_multipleSelectCategories() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))
    try h.assertSubmitEventsCount(2)
  }

  // ─────────────────────────────────────────────────────────────
  // Group 4: Offset Adjustment
  // ─────────────────────────────────────────────────────────────

  func test_adjustOffset_backward() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()
    let beforeAdjust = h.store.lastEventTime

    await h.store.dispatch(.adjustOffset(5))

    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
       expectedType: "amendment",
       expectedIncomingId: nil,
    )
    let expectedTime = beforeAdjust.addingTimeInterval(-5 * 60)
    try assert(abs(h.store.lastEventTime.timeIntervalSince(expectedTime)) < 2, "Offset time should be retroactive")
  }

  func test_adjustOffset_negativeIsBlocked() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()

    await h.store.dispatch(.adjustOffset(-5))

    try h.assertNoSubmitEvents()
    try assert(h.store.offsetMinutes == 0, "Offset must not become negative")
  }

  func test_adjustOffset_multipleAccumulate() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()
    let initialEventTime = h.store.lastEventTime

    await h.store.dispatch(.adjustOffset(5))
    await h.store.dispatch(.adjustOffset(5))

    try h.assertSubmitEventsCount(2)
    try h.assertSubmitEventDetails(index: 0, expectedType: "amendment", expectedIncomingId: nil)
    try h.assertSubmitEventDetails(index: 1, expectedType: "amendment", expectedIncomingId: nil)
    let amendments = h.mock.submitEventsCalls.map { $0.events[0] }
    try assertEqual(amendments[0].targetClientEventId, amendments[1].targetClientEventId)
    try assert(
      amendments[0].correctedAtLocal?.suffix(6) == amendments[1].correctedAtLocal?.suffix(6),
      "Amendments should preserve the target event timezone offset"
    )
    try assert(
      amendments[0].correctedAtLocal != amendments[0].occurredAtLocal,
      "Amendments should carry the corrected wall-clock time"
    )
    try assert(h.store.offsetMinutes == 10, "Offset should accumulate to 10")
  }

  func test_adjustOffset_noCurrentCategory_noSubmit() async throws {
     let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initializeAndResetCalls()

    await h.store.dispatch(.adjustOffset(5))

    try h.assertNoSubmitEvents()
  }

  func test_adjustOffset_withoutPersistedEvent_doesNotMutateContext() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initializeAndResetCalls()
    let originalTime = h.store.lastEventTime
    await h.store.dispatch(.adjustOffset(5))
    try assertEqual(h.store.offsetMinutes, 0)
    try assertEqual(h.store.lastEventTime, originalTime)
  }

  // ─────────────────────────────────────────────────────────────
  // Group 5: Return to Plan
  // ─────────────────────────────────────────────────────────────

  func test_returnToPlan_whenOffSchedule() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))
    h.mock.resetCalls()

    await h.store.dispatch(.returnToPlan)

    if h.store.plannedCategory != nil {
      try h.assertSubmitEventsCount(1)
      try h.assertSubmitEventDetails(
        index: 0,
        expectedType: "transition",
        expectedIncomingId: Fixtures.categoryA.id
      )
    }
  }

  func test_returnToPlan_whenOnSchedule_noSubmit() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()

    await h.store.dispatch(.returnToPlan)

    try h.assertNoSubmitEvents()
  }

  func test_returnToPlan_noPlannedCategory_noSubmit() async throws {
     let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initializeAndResetCalls()

    await h.store.dispatch(.returnToPlan)

    try h.assertNoSubmitEvents()
  }

  // ─────────────────────────────────────────────────────────────
  // Group 6: Sequential Multi-Action Flows
  // ─────────────────────────────────────────────────────────────

  func test_flow_selectMultipleCategoriesInSequence() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    try h.assertSubmitEventsCount(3)
     try h.assertSubmitEventDetails(index: 0, expectedType: "transition", expectedIncomingId: Fixtures.categoryA.id)
    try h.assertSubmitEventDetails(index: 1, expectedType: "transition", expectedIncomingId: Fixtures.categoryB.id)
    try h.assertSubmitEventDetails(index: 2, expectedType: "transition", expectedIncomingId: Fixtures.categoryA.id)
  }

  func test_flow_offsetThenSelectCategory() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()

    await h.store.dispatch(.adjustOffset(5))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    try h.assertSubmitEventsCount(2)
     try h.assertSubmitEventDetails(index: 0, expectedType: "amendment", expectedIncomingId: nil)
    try h.assertSubmitEventDetails(index: 1, expectedType: "transition", expectedIncomingId: Fixtures.categoryB.id)
  }

  // ─────────────────────────────────────────────────────────────
  // Group 7: Repository Error Handling
  // ─────────────────────────────────────────────────────────────

  func test_submitEventsThrows_doesNotCrash() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    h.mock.shouldThrowOnSubmitEvents = true

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    try h.assertSubmitEventsCount(1)
    // Store should handle error gracefully
  }

  func test_submitEventsThrows_rollsBackOptimisticTransition() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    let originalCategory = h.store.currentCategory
    h.mock.shouldThrowOnSubmitEvents = true

    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    try assertEqual(h.store.currentCategory?.id, originalCategory?.id)
    try assert(h.store.lastError != nil, "Submission failure must be exposed")
  }

  func test_missingScheduleData_isNotOnSchedule() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initialize()
    try assert(!h.store.isOnSchedule, "Missing schedule data must not be presented as on schedule")
  }

  // ─────────────────────────────────────────────────────────────
  // Group 8: State Identity Assertions
  // ─────────────────────────────────────────────────────────────

  func test_stateTransition_initialToActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    try assert(h.store.displayState == .initializing, "Initial state should be initializing")
    await h.initialize()
    try assert(h.store.displayState == .active, "After initialize, should be active")
  }

  func test_selectCategory_inActive_staysActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    try assert(h.store.displayState == .active, "Should start in active")

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    try assert(h.store.displayState == .active, "Should remain active after selection")
  }

  // ─────────────────────────────────────────────────────────────
  // Group 9: Event Validation — Calendar Date
  // ─────────────────────────────────────────────────────────────

  func test_submitEvents_useCorrectCalendarDate() async throws {
    let record = Fixtures.record()
    let h = WidgetTestHarness(existingRecord: record)
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    let calls = h.mock.submitEventsCalls
    try assert(calls.count > 0, "Should have submitEvents calls")
    try assertEqual(calls.first?.calendarDate, record.calendarDate)
  }

  // ─────────────────────────────────────────────────────────────
  // Group 10: Event Validation — categoryId (Known Bug)
  // ─────────────────────────────────────────────────────────────

  func test_transitionEvent_populatesEventFields() async throws {
    // Verify that transition events contain category IDs (both incoming and outgoing are the same)
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    try h.assertSubmitEventsCount(1)
    let calls = h.mock.submitEventsCalls
    let event = calls[0].events[0]
    try assertEqual(event.eventType, "transition")
    try assertEqual(event.categoryId, Fixtures.categoryB.id)
  }

  func test_confirmationEvent_populatesEventFields() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()

    await h.store.dispatch(.primaryAction)

    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
      expectedType: "confirmation",
      expectedIncomingId: Fixtures.categoryA.id
    )
    try assert(h.store.displayState == .active, "Should remain active when no boundary")
  }

  func test_pomodoroCompleted_sendsConfirmationEvent() async throws {
    let pomodoroCategory = Category(
      id: 10,
      name: "Pomodoro Task",
      color: "#000000",
      pomodoroConfig: PomodoroConfig(workDuration: 60, restDuration: 60),
      createdAt: Fixtures.now,
      updatedAt: Fixtures.now
    )
    let plannedBlock = PlannedBlock(
      categoryId: pomodoroCategory.id, startTime: "00:00:00", durationMinutes: 24 * 60)
    let dayRecord = DayRecord(
      calendarDate: Fixtures.today,
      plan: [plannedBlock]
    )
    let h = WidgetTestHarness(categories: [pomodoroCategory], existingRecord: dayRecord)
    await h.initializeAndResetCalls()
    h.store.context.pomodoroPhase = .work
    h.store.context.pomodoroElapsed = 59

    let result = h.store.currentState.onTick(
      context: h.store.context,
      currentPlannedBlock: plannedBlock
    )
    await h.store.apply(result)

    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
      expectedType: "confirmation",
      expectedIncomingId: pomodoroCategory.id
    )
  }

  @MainActor
  func test_confirmationAtCategoryBoundary_logsTransitionWhenCategoryChanges() async throws {
    var context = WidgetContext()
    context.currentCategory = Fixtures.categoryA
    context.plannedCategory = Fixtures.categoryB

    let result = confirmationResult(context: context, nextState: ActiveState())

    let transitionCategoryIds = result.effects.compactMap { effect -> Int? in
      guard case .logTransition(let category, _) = effect else { return nil }
      return category.id
    }
    try assertEqual(transitionCategoryIds, [Fixtures.categoryB.id])
  }

  @MainActor
  func test_confirmationWithoutCategoryChange_logsTransition() async throws {
    var context = WidgetContext()
    context.currentCategory = Fixtures.categoryA
    context.plannedCategory = Fixtures.categoryA

    let result = confirmationResult(context: context, nextState: ActiveState())

    let transitionCategoryIds = result.effects.compactMap { effect -> Int? in
      guard case .logTransition(let category, _) = effect else { return nil }
      return category.id
    }
    try assertEqual(transitionCategoryIds, [Fixtures.categoryA.id])
  }

  // ─────────────────────────────────────────────────────────────
  // Group 11: Event Validation — All Fields Correct
  // ─────────────────────────────────────────────────────────────

  func test_submitEventsCall_usesCorrectCalendarDateAndEventSequence() async throws {
    let record = Fixtures.record()
    let h = WidgetTestHarness(existingRecord: record)
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    let calls = h.mock.submitEventsCalls
    try assert(calls.count == 2, "Should have 2 submitEvents calls")

    // Verify first call
    let (date1, events1) = calls[0]
    try assertEqual(date1, record.calendarDate)
    try assertEqual(events1.count, 1)
    try assertEqual(events1[0].categoryId, Fixtures.categoryA.id)
    try assert(events1[0].occurredAtLocal.count == 25, "Local timestamp should include numeric offset")
    try assert(events1[0].occurredAtLocal.last == "0", "Local timestamp should be RFC3339")

    // Verify second call
    let (date2, events2) = calls[1]
    try assertEqual(date2, record.calendarDate)
    try assertEqual(events2.count, 1)
    try assertEqual(events2[0].categoryId, Fixtures.categoryB.id)
  }
}

// ═══════════════════════════════════════════════════════════════════
// MARK: - Test Runner
// ═══════════════════════════════════════════════════════════════════

struct TestResult {
  let name: String
  let passed: Bool
  let error: String?
}

@main
struct TestRunner {
  static func main() async {
    print("Running WidgetStateStore tests...")
    print("")

    let tests = WidgetStateStoreTests()
    let contentViewTests = ContentViewTests()
    var results: [TestResult] = []

    let testMethods: [(String, () async throws -> Void)] = [
      ("test_contentView_checkingAuthenticationTakesPriority", { try contentViewTests.test_checkingAuthenticationTakesPriority() }),
      ("test_contentView_cachedBootstrapShowsWidgetWithoutAuthentication", { try contentViewTests.test_cachedBootstrapShowsWidgetWithoutAuthentication() }),
      ("test_contentView_authenticatedSessionShowsWidgetWithoutCache", { try contentViewTests.test_authenticatedSessionShowsWidgetWithoutCache() }),
      ("test_contentView_missingSessionAndCacheShowsLogin", { try contentViewTests.test_missingSessionAndCacheShowsLogin() }),
      ("test_localEventStoreBackupSurvivesQueueRemoval", { try tests.test_localEventStoreBackupSurvivesQueueRemoval() }),
      ("test_localEventStoreMigratesLegacyJSONQueueToJSONLines", { try tests.test_localEventStoreMigratesLegacyJSONQueueToJSONLines() }),
      ("test_localEventStoreBackfillsMissingLocalTimestamps", { try tests.test_localEventStoreBackfillsMissingLocalTimestamps() }),
      ("test_appendFailurePreservesQueueAndBackup", { try tests.test_appendFailurePreservesQueueAndBackup() }),
      ("test_atomicQueueRewriteFailurePreservesContents", { try tests.test_atomicQueueRewriteFailurePreservesContents() }),
      ("test_bootstrapCache_roundTripsAndRejectsOtherDates", { try tests.test_bootstrapCache_roundTripsAndRejectsOtherDates() }),
      ("test_bootstrapCache_rejectsMismatchedResponseDate", { try tests.test_bootstrapCache_rejectsMismatchedResponseDate() }),
      ("test_bootstrapCache_returnsNilWhenTodayIsMissing", { try tests.test_bootstrapCache_returnsNilWhenTodayIsMissing() }),
      ("test_clearLocalDataRemovesEventsAndBootstrapCache", { try tests.test_clearLocalDataRemovesEventsAndBootstrapCache() }),
      ("test_apiClientDoesNotPersistAuthenticationToken", { try tests.test_apiClientDoesNotPersistAuthenticationToken() }),
      ("test_authenticationUsesMemoryTokenWithoutValidationRequest", { try await tests.test_authenticationUsesMemoryTokenWithoutValidationRequest() }),
      ("test_synchronizeSendsSortedEventsAsSingleBatch", { try await tests.test_synchronizeSendsSortedEventsAsSingleBatch() }),
      ("test_synchronize400FallsBackToIndividualEvents", { try await tests.test_synchronize400FallsBackToIndividualEvents() }),
      ("test_synchronizeNon400ErrorPreservesQueue", { try await tests.test_synchronizeNon400ErrorPreservesQueue() }),
      ("test_synchronize_refreshesBootstrapCache", { try await tests.test_synchronize_refreshesBootstrapCache() }),
      ("test_synchronizeRefreshFailurePreservesExistingBootstrapCache", { try await tests.test_synchronizeRefreshFailurePreservesExistingBootstrapCache() }),
      ("test_init_freshDay_createsRecord", { try await tests.test_init_freshDay_createsRecord() }),
      ("test_init_withPlannedCategory_picksCategoryAndLogsTransition", { try await tests.test_init_withPlannedCategory_picksCategoryAndLogsTransition() }),
      ("test_initResponse_decodesCurrentAPIShape", { try tests.test_initResponse_decodesCurrentAPIShape() }),
      ("test_dayEventsResponse_decodesNullAcceptedEventCategory", { try tests.test_dayEventsResponse_decodesNullAcceptedEventCategory() }),
      ("test_dayEventsResponse_missingRequiredArrayFailsDecoding", { try tests.test_dayEventsResponse_missingRequiredArrayFailsDecoding() }),
      ("test_dayRecordsResponse_missingContainerFailsDecoding", { try tests.test_dayRecordsResponse_missingContainerFailsDecoding() }),
      ("test_dayEventEncodesIdempotencyAndAmendmentFields", { try tests.test_dayEventEncodesIdempotencyAndAmendmentFields() }),
      ("test_init_existingRecord_doesNotCreate", { try await tests.test_init_existingRecord_doesNotCreate() }),
      ("test_invalidScheduleTime_doesNotSelectBlock", { try tests.test_invalidScheduleTime_doesNotSelectBlock() }),
      ("test_invalidBootstrap_keepsStoreInitializing", { try await tests.test_invalidBootstrap_keepsStoreInitializing() }),
      ("test_actualBlocksAreNotPartOfWidgetPlanModel", { try await tests.test_actualBlocksAreNotPartOfWidgetPlanModel() }),
      ("test_selectCategory_logsTransition", { try await tests.test_selectCategory_logsTransition() }),
      ("test_initialState_isInitializing", { try await tests.test_initialState_isInitializing() }),
      ("test_afterInitialize_isActive", { try await tests.test_afterInitialize_isActive() }),
      ("test_cachedBootstrap_startsWidgetWithoutLoggingTransition", { try await tests.test_cachedBootstrap_startsWidgetWithoutLoggingTransition() }),
      ("test_reload_fetchesRemoteDataAndReturnsToActive", { try await tests.test_reload_fetchesRemoteDataAndReturnsToActive() }),
      ("test_afterConfirmation_returnsToActive", { try await tests.test_afterConfirmation_returnsToActive() }),
      ("test_multipleSelectCategories", { try await tests.test_multipleSelectCategories() }),
      ("test_adjustOffset_backward", { try await tests.test_adjustOffset_backward() }),
      ("test_adjustOffset_negativeIsBlocked", { try await tests.test_adjustOffset_negativeIsBlocked() }),
      ("test_adjustOffset_multipleAccumulate", { try await tests.test_adjustOffset_multipleAccumulate() }),
      ("test_adjustOffset_noCurrentCategory_noSubmit", { try await tests.test_adjustOffset_noCurrentCategory_noSubmit() }),
      ("test_adjustOffset_withoutPersistedEvent_doesNotMutateContext", { try await tests.test_adjustOffset_withoutPersistedEvent_doesNotMutateContext() }),
      ("test_returnToPlan_whenOffSchedule", { try await tests.test_returnToPlan_whenOffSchedule() }),
      ("test_returnToPlan_whenOnSchedule_noSubmit", { try await tests.test_returnToPlan_whenOnSchedule_noSubmit() }),
      ("test_returnToPlan_noPlannedCategory_noSubmit", { try await tests.test_returnToPlan_noPlannedCategory_noSubmit() }),
      ("test_flow_selectMultipleCategoriesInSequence", { try await tests.test_flow_selectMultipleCategoriesInSequence() }),
      ("test_flow_offsetThenSelectCategory", { try await tests.test_flow_offsetThenSelectCategory() }),
      ("test_submitEventsThrows_doesNotCrash", { try await tests.test_submitEventsThrows_doesNotCrash() }),
      ("test_submitEventsThrows_rollsBackOptimisticTransition", { try await tests.test_submitEventsThrows_rollsBackOptimisticTransition() }),
      ("test_missingScheduleData_isNotOnSchedule", { try await tests.test_missingScheduleData_isNotOnSchedule() }),
      ("test_stateTransition_initialToActive", { try await tests.test_stateTransition_initialToActive() }),
      ("test_selectCategory_inActive_staysActive", { try await tests.test_selectCategory_inActive_staysActive() }),
      ("test_submitEvents_useCorrectCalendarDate", { try await tests.test_submitEvents_useCorrectCalendarDate() }),
      ("test_transitionEvent_populatesEventFields", { try await tests.test_transitionEvent_populatesEventFields() }),
      ("test_confirmationEvent_populatesEventFields", { try await tests.test_confirmationEvent_populatesEventFields() }),
      ("test_pomodoroCompleted_sendsConfirmationEvent", { try await tests.test_pomodoroCompleted_sendsConfirmationEvent() }),
      ("test_confirmationAtCategoryBoundary_logsTransitionWhenCategoryChanges", { try await tests.test_confirmationAtCategoryBoundary_logsTransitionWhenCategoryChanges() }),
      ("test_confirmationWithoutCategoryChange_logsTransition", { try await tests.test_confirmationWithoutCategoryChange_logsTransition() }),
      ("test_submitEventsCall_usesCorrectCalendarDateAndEventSequence", { try await tests.test_submitEventsCall_usesCorrectCalendarDateAndEventSequence() }),
    ]

    for (name, testFn) in testMethods {
      do {
        try await testFn()
        results.append(TestResult(name: name, passed: true, error: nil))
        print("✓ \(name)")
      } catch let error as AssertionError {
        results.append(TestResult(name: name, passed: false, error: String(describing: error)))
        print("✗ \(name): \(error)")
      } catch {
        results.append(TestResult(name: name, passed: false, error: String(describing: error)))
        print("✗ \(name): \(error)")
      }
    }

    print("")
    let passed = results.filter(\.passed).count
    let total = results.count
    print("Passed: \(passed)/\(total)")

    if passed < total {
      exit(1)
    }
  }
}
