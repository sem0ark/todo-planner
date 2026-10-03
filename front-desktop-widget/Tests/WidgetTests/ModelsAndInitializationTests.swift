import Foundation

@MainActor
final class ModelsAndInitializationTests: WidgetTestCase {

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

  func test_initialState_isInitializing() throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    try assert(h.store.displayState == .initializing, "Initial state should be initializing")
  }

  func test_afterInitialize_isActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initialize()
    try assert(h.store.displayState == .active, "After init, state should be active")
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

  func test_missingScheduleData_isNotOnSchedule() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initialize()
    try assert(!h.store.isOnSchedule, "Missing schedule data must not be presented as on schedule")
  }

  func test_stateTransition_initialToActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    try assert(h.store.displayState == .initializing, "Initial state should be initializing")
    await h.initialize()
    try assert(h.store.displayState == .active, "After initialize, should be active")
  }

  func test_startWithoutCacheOrAuthenticationShowsLogin() async throws {
    let harness = WidgetTestHarness(existingRecord: nil)
    harness.mock.authToken = nil

    await harness.store.start()

    try assertEqual(harness.store.screenState, .login)
  }

  func test_startWithCachedBootstrapWithoutAuthenticationShowsWidget() async throws {
    let harness = WidgetTestHarness(existingRecord: Fixtures.record())
    await harness.initialize()
    harness.mock.authToken = nil
    harness.store.screenState = .checkingAuthentication

    await harness.store.start()

    try assertEqual(harness.store.screenState, .widget)
    try assertEqual(harness.store.displayState, .active)
  }

  func test_startWithAuthenticationWithoutCacheLoadsAPIAndShowsWidget() async throws {
    let harness = WidgetTestHarness(existingRecord: Fixtures.record())

    await harness.store.start()

    try assertEqual(harness.store.screenState, .widget)
    try assertEqual(harness.store.displayState, .active)
    try assertEqual(harness.mock.calls.count, 1)
    try assertEqual(harness.mock.calls.compactMap { call -> String? in
      guard case .initialize(let date) = call else { return nil }
      return date
    }, [Fixtures.today])
  }

  func test_authenticationSucceededLoadsFreshAPIDataWithoutUsingCache() async throws {
    let harness = WidgetTestHarness(existingRecord: Fixtures.record())
    await harness.initialize()
    harness.store.screenState = .login
    harness.mock.resetCalls()

    await harness.store.authenticationSucceeded()

    try assertEqual(harness.store.screenState, .widget)
    try assertEqual(harness.store.displayState, .active)
    try assertEqual(harness.mock.calls.count, 1)
    try assertEqual(harness.mock.calls.first?.description, "initialize(\(Fixtures.today))")
  }

  func test_authenticationSucceededWithNoAPIDataShowsError() async throws {
    let harness = WidgetTestHarness(existingRecord: nil)
    harness.store.screenState = .login

    await harness.store.authenticationSucceeded()

    try assert(
      isErrorState(harness.store.screenState),
      "API bootstrap failure should leave the loading state")
    try assert(harness.store.lastError != nil, "API bootstrap failure should be exposed")
  }

  func test_startWithAuthenticationIgnoresCachedBootstrap() async throws {
    let harness = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await harness.initialize()
    harness.mock.resetCalls()

    harness.store.screenState = .checkingAuthentication
    await harness.store.start()

    try assertEqual(harness.store.screenState, .widget)
    try assertEqual(harness.mock.calls.count, 1)
    try assertEqual(harness.mock.calls.first?.description, "initialize(\(Fixtures.today))")
  }

  func test_startWithAuthenticationFailureShowsError() async throws {
    let harness = WidgetTestHarness(existingRecord: nil)
    harness.mock.initializationError = StorageError.networkFailure(
      NSError(domain: "MockError", code: 1))

    await harness.store.start()

    try assert(isErrorState(harness.store.screenState), "Remote failure should show an error state")
    try assert(harness.store.lastError != nil, "Remote failure should be exposed")
  }

  private func isErrorState(_ state: WidgetScreenState) -> Bool {
    if case .error = state { return true }
    return false
  }

  static func testMethods() -> [TestCase] {
    let tests = ModelsAndInitializationTests()
    return [
      ("test_initResponse_decodesCurrentAPIShape", { try tests.test_initResponse_decodesCurrentAPIShape() }),
      ("test_dayEventsResponse_decodesNullAcceptedEventCategory", { try tests.test_dayEventsResponse_decodesNullAcceptedEventCategory() }),
      ("test_dayEventsResponse_missingRequiredArrayFailsDecoding", { try tests.test_dayEventsResponse_missingRequiredArrayFailsDecoding() }),
      ("test_dayRecordsResponse_missingContainerFailsDecoding", { try tests.test_dayRecordsResponse_missingContainerFailsDecoding() }),
      ("test_dayEventEncodesIdempotencyAndAmendmentFields", { try tests.test_dayEventEncodesIdempotencyAndAmendmentFields() }),
      ("test_init_freshDay_createsRecord", { try await tests.test_init_freshDay_createsRecord() }),
      ("test_init_withPlannedCategory_picksCategoryAndLogsTransition", { try await tests.test_init_withPlannedCategory_picksCategoryAndLogsTransition() }),
      ("test_init_existingRecord_doesNotCreate", { try await tests.test_init_existingRecord_doesNotCreate() }),
      ("test_invalidScheduleTime_doesNotSelectBlock", { try tests.test_invalidScheduleTime_doesNotSelectBlock() }),
      ("test_invalidBootstrap_keepsStoreInitializing", { try await tests.test_invalidBootstrap_keepsStoreInitializing() }),
      ("test_actualBlocksAreNotPartOfWidgetPlanModel", { try await tests.test_actualBlocksAreNotPartOfWidgetPlanModel() }),
      ("test_initialState_isInitializing", { try tests.test_initialState_isInitializing() }),
      ("test_afterInitialize_isActive", { try await tests.test_afterInitialize_isActive() }),
      ("test_cachedBootstrap_startsWidgetWithoutLoggingTransition", { try await tests.test_cachedBootstrap_startsWidgetWithoutLoggingTransition() }),
      ("test_reload_fetchesRemoteDataAndReturnsToActive", { try await tests.test_reload_fetchesRemoteDataAndReturnsToActive() }),
      ("test_missingScheduleData_isNotOnSchedule", { try await tests.test_missingScheduleData_isNotOnSchedule() }),
      ("test_stateTransition_initialToActive", { try await tests.test_stateTransition_initialToActive() }),
      ("test_startWithoutCacheOrAuthenticationShowsLogin", { try await tests.test_startWithoutCacheOrAuthenticationShowsLogin() }),
      ("test_startWithCachedBootstrapWithoutAuthenticationShowsWidget", { try await tests.test_startWithCachedBootstrapWithoutAuthenticationShowsWidget() }),
      ("test_startWithAuthenticationWithoutCacheLoadsAPIAndShowsWidget", { try await tests.test_startWithAuthenticationWithoutCacheLoadsAPIAndShowsWidget() }),
      ("test_authenticationSucceededLoadsFreshAPIDataWithoutUsingCache", { try await tests.test_authenticationSucceededLoadsFreshAPIDataWithoutUsingCache() }),
      ("test_authenticationSucceededWithNoAPIDataShowsError", { try await tests.test_authenticationSucceededWithNoAPIDataShowsError() }),
      ("test_startWithAuthenticationIgnoresCachedBootstrap", { try await tests.test_startWithAuthenticationIgnoresCachedBootstrap() }),
      ("test_startWithAuthenticationFailureShowsError", { try await tests.test_startWithAuthenticationFailureShowsError() }),
    ]
  }
}
