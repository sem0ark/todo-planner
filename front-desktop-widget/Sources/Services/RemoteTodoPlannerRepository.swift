import Foundation

final class RemoteTodoPlannerRepository: @unchecked Sendable, TodoPlannerRepository {
  private let api: TodoPlannerAPI
  private let eventStore: LocalEventStore
  private let bootstrapCache: BootstrapCache

  init(
    api: TodoPlannerAPI = APIClient.shared,
    eventStore: LocalEventStore = .shared,
    bootstrapCache: BootstrapCache = .shared
  ) {
    self.api = api
    self.eventStore = eventStore
    self.bootstrapCache = bootstrapCache
  }

  func getAuthToken() -> String? {
    return api.authToken
  }

  func persistAuthToken(_ token: String) async throws {
    api.setAuthToken(token)
  }

  func clearAuth() async throws {
    api.clearAuthToken()
  }

  func validateAuth() async throws -> Bool {
    return try await api.validateToken()
  }

  func initialize(calendarDate: String) async throws -> InitResponse {
    let response = try await api.initialize(calendarDate: calendarDate)
    try bootstrapCache.save(response: response, calendarDate: calendarDate)
    return response
  }

  func cachedInitialization(calendarDate: String) throws -> InitResponse? {
    try bootstrapCache.load(calendarDate: calendarDate)?.response
  }

  func submitEvents(calendarDate: String, events: [DayEvent]) async throws -> DayEventsResponse {
    for event in events {
      try eventStore.append(calendarDate: calendarDate, event: event)
    }
    return DayEventsResponse(calendarDate: calendarDate)
  }

  func hasPendingSync() async -> Bool {
    (try? !eventStore.pendingEvents().isEmpty) ?? false
  }

  func synchronize() async throws {
    WidgetLogger.debug("Synchronization started")
    var synchronizationError: Error?
    do {
      try await synchronizePendingEvents()
    } catch {
      synchronizationError = error
    }

    if api.authToken != nil {
      WidgetLogger.debug("Refreshing bootstrap during synchronization")
      do {
        _ = try await initialize(calendarDate: DateFormatter.yyyyMMdd.string(from: Date()))
      } catch {
        if synchronizationError == nil {
          synchronizationError = error
        }
      }
    }

    if let synchronizationError {
      WidgetLogger.debug(
        "Synchronization failed", context: ["error": String(describing: synchronizationError)])
      throw synchronizationError
    }
    WidgetLogger.debug("Synchronization completed")
  }

  func clearLocalData() throws {
    try eventStore.clearAll()
    try bootstrapCache.removeAll()
  }

  private func synchronizePendingEvents() async throws {
    let pendingEvents = try eventStore.pendingEvents()
    let groupedEvents = Dictionary(grouping: pendingEvents, by: \.calendarDate)

    for (calendarDate, entries) in groupedEvents {
      let events = entries.map(\.event).sorted { firstEvent, secondEvent in
        firstEvent.occurredAt < secondEvent.occurredAt
      }

      do {
        let response = try await api.postDayEvents(date: calendarDate, events: events)
        let acknowledgedIds = Set(response.acceptedEvents.map(\.clientEventId))
          .union(response.duplicateClientEventIds)
        try eventStore.remove(clientEventIds: acknowledgedIds)
      } catch APIError.serverError(400, let message) {
        WidgetLogger.error(
          "Batch event synchronization failed; retrying individually",
          context: ["calendarDate": calendarDate, "error": message])

        var removableEventIds = Set<String>()
        for event in events {
          do {
            let response = try await api.postDayEvents(date: calendarDate, events: [event])
            removableEventIds.formUnion(response.acceptedEvents.map(\.clientEventId))
            removableEventIds.formUnion(response.duplicateClientEventIds)
          } catch APIError.serverError(400, let individualMessage) {
            WidgetLogger.error(
              "Skipping rejected event during synchronization",
              context: [
                "calendarDate": calendarDate,
                "clientEventId": event.clientEventId,
                "error": individualMessage,
              ])
            removableEventIds.insert(event.clientEventId)
          }
        }

        try eventStore.remove(clientEventIds: removableEventIds)
      }
    }
  }
}
