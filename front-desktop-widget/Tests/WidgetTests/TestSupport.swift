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
  var authToken: String? = "test-token"
  var stubbedCategories: [Category] = []
  var stubbedDayRecord: DayRecord? = nil
  var stubbedCreatedRecord: DayRecord?
  var stubbedEventsResponse: DayEventsResponse?
  var shouldThrowOnSubmitEvents = false
  var initializationError: Error?
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

  func getAuthToken() -> String? { authToken }
  func persistAuthToken(_ token: String) async throws {}
  func clearAuth() async throws {}
  func validateAuth() async throws -> Bool {
    validateAuthCallCount += 1
    return true
  }

  func initialize(calendarDate: String) async throws -> InitResponse {
    calls.append(.initialize(date: calendarDate))
    if let initializationError {
      throw initializationError
    }
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

@MainActor
class WidgetTestCase {
  func makeTemporaryEventStore() throws -> (LocalEventStore, URL) {
    let directory = FileManager.default.temporaryDirectory
      .appendingPathComponent("TodoPlannerWidgetTests-\(UUID().uuidString)")
    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    return (LocalEventStore(storageDirectory: directory), directory)
  }

  func makeTemporaryBootstrapCache() throws -> (BootstrapCache, URL) {
    let directory = FileManager.default.temporaryDirectory
      .appendingPathComponent("TodoPlannerWidgetBootstrapTests-\(UUID().uuidString)")
    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    return (BootstrapCache(storageDirectory: directory), directory)
  }

  func makeBootstrapResponse(calendarDate: String = Fixtures.today) -> InitResponse {
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
}


typealias TestCase = (String, () async throws -> Void)
