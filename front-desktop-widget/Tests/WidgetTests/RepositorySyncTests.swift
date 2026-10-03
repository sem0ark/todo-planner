import Foundation

@MainActor
final class RepositorySyncTests: WidgetTestCase {
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

  func test_synchronizeWithoutAuthenticationRequestsLogin() async throws {
    let harness = WidgetTestHarness(existingRecord: Fixtures.record())
    harness.mock.authToken = nil
    var authenticationRequired = false
    let observer = NotificationCenter.default.addObserver(
      forName: .authenticationRequired,
      object: nil,
      queue: .main
    ) { _ in
      authenticationRequired = true
    }

    await harness.store.synchronize()
    NotificationCenter.default.removeObserver(observer)

    try assert(authenticationRequired, "Synchronization should request authentication")
    try assert(harness.store.lastError == "unauthorized", "Missing authentication should be exposed")
  }

  static func testMethods() -> [TestCase] {
    let tests = RepositorySyncTests()
    return [
      ("test_authenticationUsesMemoryTokenWithoutValidationRequest", { try await tests.test_authenticationUsesMemoryTokenWithoutValidationRequest() }),
      ("test_apiClientDoesNotPersistAuthenticationToken", { try tests.test_apiClientDoesNotPersistAuthenticationToken() }),
      ("test_synchronizeSendsSortedEventsAsSingleBatch", { try await tests.test_synchronizeSendsSortedEventsAsSingleBatch() }),
      ("test_synchronize400FallsBackToIndividualEvents", { try await tests.test_synchronize400FallsBackToIndividualEvents() }),
      ("test_synchronizeNon400ErrorPreservesQueue", { try await tests.test_synchronizeNon400ErrorPreservesQueue() }),
      ("test_synchronize_refreshesBootstrapCache", { try await tests.test_synchronize_refreshesBootstrapCache() }),
      ("test_synchronizeRefreshFailurePreservesExistingBootstrapCache", { try await tests.test_synchronizeRefreshFailurePreservesExistingBootstrapCache() }),
      ("test_synchronizeWithoutAuthenticationRequestsLogin", { try await tests.test_synchronizeWithoutAuthenticationRequestsLogin() }),
    ]
  }
}
