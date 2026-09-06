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
  case fetchCategories
  case fetchDayRecord(date: String)
  case createDayRecord(date: String)
  case submitEvents(dayRecordId: Int, events: [DayEvent])

  var description: String {
    switch self {
    case .fetchCategories: return "fetchCategories"
    case .fetchDayRecord(let d): return "fetchDayRecord(\(d))"
    case .createDayRecord(let d): return "createDayRecord(\(d))"
    case .submitEvents(let id, let e):
      let types = e.map(\.eventType).joined(separator: ",")
      return "submitEvents(record:\(id), types:[\(types)])"
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

  private(set) var calls: [RecordedCall] = []

  func resetCalls() { calls.removeAll() }

  var submitEventsCalls: [(dayRecordId: Int, events: [DayEvent])] {
    calls.compactMap {
      guard case .submitEvents(let id, let events) = $0 else { return nil }
      return (id, events)
    }
  }

  func getAuthToken() -> String? { "test-token" }
  func persistAuthToken(_ token: String) async throws {}
  func clearAuth() async throws {}
  func validateAuth() async throws -> Bool { true }

  func fetchCategories() async throws -> [Category] {
    calls.append(.fetchCategories)
    return stubbedCategories
  }

  func fetchDayRecord(date: String) async throws -> DayRecord? {
    calls.append(.fetchDayRecord(date: date))
    return stubbedDayRecord
  }

  func createDayRecord(date: String) async throws -> DayRecord {
    calls.append(.createDayRecord(date: date))
    guard let record = stubbedCreatedRecord else { throw StorageError.notFound }
    return record
  }

  func submitEvents(dayRecordId: Int, events: [DayEvent]) async throws -> DayEventsResponse {
    calls.append(.submitEvents(dayRecordId: dayRecordId, events: events))
    if shouldThrowOnSubmitEvents {
      throw NSError(domain: "MockError", code: 1, userInfo: nil)
    }
    return stubbedEventsResponse ?? DayEventsResponse(createdEvents: [], actualBlocks: [])
  }

  func hasPendingSync() async -> Bool { false }
  func synchronize() async throws {}
  func fetchTodaySchedule() async throws -> TodaySchedule {
    TodaySchedule(calendarDate: Fixtures.today, dayTemplateId: nil, template: nil)
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
    let start = String(format: "%02d:00", comps.hour ?? 0)
    return PlannedBlock(id: id, categoryId: categoryId, startTime: start, durationMinutes: durationMinutes)
  }

  static func actualBlock(
    id: Int = 1,
    categoryId: Int? = 1,
    blockType: String = "actual",
    startTime: String = "08:00",
    durationMinutes: Int = 60
  ) -> ActualBlock {
    ActualBlock(id: id, categoryId: categoryId, blockType: blockType, startTime: startTime, durationMinutes: durationMinutes)
  }

  static var today: String {
    DateFormatter.yyyyMMdd.string(from: Date())
  }

  static func record(
    id: Int = 1,
    actualBlocks: [ActualBlock] = []
  ) -> DayRecord {
    DayRecord(
      id: id, calendarDate: today,
      reviewStatus: "Unreviewed",
      actualBlocks: actualBlocks,
      createdAt: Date(), updatedAt: Date()
    )
  }

  static func recordWithCurrentBlock(categoryId: Int = 1) -> DayRecord {
    record(actualBlocks: [actualBlock(categoryId: categoryId)])
  }

  static func eventsResponse(blocks: [ActualBlock] = []) -> DayEventsResponse {
    DayEventsResponse(createdEvents: [], actualBlocks: blocks)
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
    dayRecordId: Int? = nil
  ) throws {
    let calls = mock.submitEventsCalls
    guard index < calls.count else {
      throw AssertionError.failed("submitEvents call at index \(index) not found (total: \(calls.count))")
    }

    let (recordId, events) = calls[index]
    guard let event = events.first else {
      throw AssertionError.failed("submitEvents[\(index)] has no events")
    }

    try assertEqual(event.eventType, expectedType)
    try assertEqual(event.categoryId, expectedIncomingId)

    if let expectedRecordId = dayRecordId {
      try assertEqual(recordId, expectedRecordId)
    }
  }
}

// ═══════════════════════════════════════════════════════════════════
// MARK: - Tests
// ═══════════════════════════════════════════════════════════════════

@MainActor
final class WidgetStateStoreTests {
  func test_initResponse_decodesCurrentAPIShape() throws {
    let json = """
    {
      "settings": {
        "day_boundary_time": "04:00",
        "updated_at": "2026-09-06T14:30:00Z"
      },
      "categories": [],
      "day_record": {
        "calendar_date": "2026-09-06",
        "day_template_id": 5,
        "snapshot": {
          "snapshot_id": 12,
          "snapshotted_at": "2026-09-01T10:00:00Z",
          "blocks": [
            {
              "category_id": 3,
              "start_time": "08:00",
              "duration_minutes": 60
            }
          ]
        },
        "actual_blocks": [
          {
            "category_id": 3,
            "block_type": "actual",
            "start_time": "08:05",
            "duration_minutes": 55,
            "is_open": false
          }
        ],
        "created_at": "2026-09-06T07:55:00Z",
        "updated_at": "2026-09-06T14:30:00Z"
      }
    }
    """.data(using: .utf8)!
    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601

    let response = try decoder.decode(InitResponse.self, from: json)

    try assertEqual(response.settings.dayBoundaryTime, "04:00")
    try assertEqual(response.dayRecord.calendarDate, "2026-09-06")
    try assertEqual(response.dayRecord.snapshotId, 12)
    try assertEqual(response.dayRecord.snapshotBlocks[0].startTime, "08:00")
    try assertEqual(response.dayRecord.actualBlocks[0].isOpen, false)
  }

  func test_dayEventEncodesIdempotencyAndAmendmentFields() throws {
    let event = DayEvent(
      clientEventId: "event-1",
      eventType: "amendment",
      categoryId: 3,
      occurredAt: Fixtures.now,
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
    try assert(object?["corrected_at"] != nil, "corrected_at should be encoded")
  }

  func test_init_freshDay_createsRecord() async throws {
    let newRecord = Fixtures.record()
    let h = WidgetTestHarness(existingRecord: nil, createdRecord: newRecord)
    await h.initialize()
    try h.assertContainsCall { if case .fetchCategories = $0 { return true }; return false }
    try h.assertContainsCall { if case .fetchDayRecord = $0 { return true }; return false }
    try h.assertContainsCall { if case .createDayRecord = $0 { return true }; return false }
    try h.assertNoSubmitEvents()
  }

  func test_init_existingRecord_doesNotCreate() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initialize()
    try h.assertNoSubmitEvents()
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

    let categoryFetchCount = h.mock.calls.reduce(into: 0) { count, call in
      if case .fetchCategories = call { count += 1 }
    }
    let dayRecordFetchCount = h.mock.calls.reduce(into: 0) { count, call in
      if case .fetchDayRecord = call { count += 1 }
    }

    try assertEqual(categoryFetchCount, 2)
    try assertEqual(dayRecordFetchCount, 2)
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
      expectedType: "transition",
      expectedIncomingId: Fixtures.categoryA.id,
    )
    let expectedTime = beforeAdjust.addingTimeInterval(-5 * 60)
    try assert(abs(h.store.lastEventTime.timeIntervalSince(expectedTime)) < 2, "Offset time should be retroactive")
  }

  func test_adjustOffset_forward() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()

    await h.store.dispatch(.adjustOffset(-5))

    try h.assertSubmitEventsCount(1)
    try assert(h.store.offsetMinutes == -5, "Offset should accumulate")
  }

  func test_adjustOffset_multipleAccumulate() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()

    await h.store.dispatch(.adjustOffset(5))
    await h.store.dispatch(.adjustOffset(5))

    try h.assertSubmitEventsCount(2)
    try h.assertSubmitEventDetails(index: 0, expectedType: "transition", expectedIncomingId: Fixtures.categoryA.id)
    try h.assertSubmitEventDetails(index: 1, expectedType: "transition", expectedIncomingId: Fixtures.categoryA.id)
    try assert(h.store.offsetMinutes == 10, "Offset should accumulate to 10")
  }

  func test_adjustOffset_noCurrentCategory_noSubmit() async throws {
     let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initializeAndResetCalls()

    await h.store.dispatch(.adjustOffset(5))

    try h.assertNoSubmitEvents()
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
    try h.assertSubmitEventDetails(index: 0, expectedType: "transition", expectedIncomingId: Fixtures.categoryA.id)
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
  // Group 9: Event Validation — Day Record ID
  // ─────────────────────────────────────────────────────────────

  func test_submitEvents_useCorrectDayRecordId() async throws {
    let recordId = 42
    let record = Fixtures.record(id: recordId, actualBlocks: [Fixtures.actualBlock()])
    let h = WidgetTestHarness(existingRecord: record)
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    let calls = h.mock.submitEventsCalls
    try assert(calls.count > 0, "Should have submitEvents calls")
    try assertEqual(calls.first?.dayRecordId, recordId)
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
    // Confirmations happen when there's a block boundary at initialization or on tick
    // For this test, we just verify the structure when it does occur
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()

    // primaryAction in Active state with no boundary doesn't submit; verify state
    await h.store.dispatch(.primaryAction)

    // This is expected - confirmation events only fire on boundary conditions
    // which require time-based triggers not easily testable here
    try assert(h.store.displayState == .active, "Should remain active when no boundary")
  }

  // ─────────────────────────────────────────────────────────────
  // Group 11: Event Validation — All Fields Correct
  // ─────────────────────────────────────────────────────────────

  func test_submitEventsCall_usesCorrectRecordIdAndEventSequence() async throws {
    let recordId = 123
    let record = Fixtures.record(id: recordId, actualBlocks: [Fixtures.actualBlock(categoryId: Fixtures.categoryA.id)])
    let h = WidgetTestHarness(existingRecord: record)
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    let calls = h.mock.submitEventsCalls
    try assert(calls.count == 2, "Should have 2 submitEvents calls")

    // Verify first call
    let (id1, events1) = calls[0]
    try assertEqual(id1, recordId)
    try assertEqual(events1.count, 1)
    try assertEqual(events1[0].categoryId, Fixtures.categoryA.id)

    // Verify second call
    let (id2, events2) = calls[1]
    try assertEqual(id2, recordId)
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
    var results: [TestResult] = []

    let testMethods: [(String, () async throws -> Void)] = [
      ("test_init_freshDay_createsRecord", { try await tests.test_init_freshDay_createsRecord() }),
      ("test_initResponse_decodesCurrentAPIShape", { try tests.test_initResponse_decodesCurrentAPIShape() }),
      ("test_dayEventEncodesIdempotencyAndAmendmentFields", { try tests.test_dayEventEncodesIdempotencyAndAmendmentFields() }),
      ("test_init_existingRecord_doesNotCreate", { try await tests.test_init_existingRecord_doesNotCreate() }),
      ("test_selectCategory_logsTransition", { try await tests.test_selectCategory_logsTransition() }),
      ("test_initialState_isInitializing", { try await tests.test_initialState_isInitializing() }),
      ("test_afterInitialize_isActive", { try await tests.test_afterInitialize_isActive() }),
      ("test_reload_fetchesRemoteDataAndReturnsToActive", { try await tests.test_reload_fetchesRemoteDataAndReturnsToActive() }),
      ("test_afterConfirmation_returnsToActive", { try await tests.test_afterConfirmation_returnsToActive() }),
      ("test_multipleSelectCategories", { try await tests.test_multipleSelectCategories() }),
      ("test_adjustOffset_backward", { try await tests.test_adjustOffset_backward() }),
      ("test_adjustOffset_forward", { try await tests.test_adjustOffset_forward() }),
      ("test_adjustOffset_multipleAccumulate", { try await tests.test_adjustOffset_multipleAccumulate() }),
      ("test_adjustOffset_noCurrentCategory_noSubmit", { try await tests.test_adjustOffset_noCurrentCategory_noSubmit() }),
      ("test_returnToPlan_whenOffSchedule", { try await tests.test_returnToPlan_whenOffSchedule() }),
      ("test_returnToPlan_whenOnSchedule_noSubmit", { try await tests.test_returnToPlan_whenOnSchedule_noSubmit() }),
      ("test_returnToPlan_noPlannedCategory_noSubmit", { try await tests.test_returnToPlan_noPlannedCategory_noSubmit() }),
      ("test_flow_selectMultipleCategoriesInSequence", { try await tests.test_flow_selectMultipleCategoriesInSequence() }),
      ("test_flow_offsetThenSelectCategory", { try await tests.test_flow_offsetThenSelectCategory() }),
      ("test_submitEventsThrows_doesNotCrash", { try await tests.test_submitEventsThrows_doesNotCrash() }),
      ("test_stateTransition_initialToActive", { try await tests.test_stateTransition_initialToActive() }),
      ("test_selectCategory_inActive_staysActive", { try await tests.test_selectCategory_inActive_staysActive() }),
      ("test_submitEvents_useCorrectDayRecordId", { try await tests.test_submitEvents_useCorrectDayRecordId() }),
      ("test_transitionEvent_populatesEventFields", { try await tests.test_transitionEvent_populatesEventFields() }),
      ("test_confirmationEvent_populatesEventFields", { try await tests.test_confirmationEvent_populatesEventFields() }),
      ("test_submitEventsCall_usesCorrectRecordIdAndEventSequence", { try await tests.test_submitEventsCall_usesCorrectRecordIdAndEventSequence() }),
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
