import Foundation

final class RemoteTodoPlannerRepository: @unchecked Sendable, TodoPlannerRepository {
  private let api = APIClient.shared

  func getAuthToken() -> String? {
    return UserDefaults.standard.string(forKey: "com.todoplanner.widget.jwt_token")
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
    return try await api.initialize(calendarDate: calendarDate)
  }

  func submitEvents(calendarDate: String, events: [DayEvent]) async throws -> DayEventsResponse {
    for event in events {
      try LocalEventStore.shared.append(calendarDate: calendarDate, event: event)
    }
    return DayEventsResponse(calendarDate: calendarDate)
  }

  func hasPendingSync() async -> Bool {
    (try? !LocalEventStore.shared.pendingEvents().isEmpty) ?? false
  }

  func synchronize() async throws {
    let pendingEvents = try LocalEventStore.shared.pendingEvents()
    let groupedEvents = Dictionary(grouping: pendingEvents, by: \.calendarDate)

    for (calendarDate, entries) in groupedEvents {
      let response = try await api.postDayEvents(
        date: calendarDate,
        events: entries.map(\.event)
      )
      let acknowledgedIds = Set(response.acceptedEvents.map(\.clientEventId))
        .union(response.duplicateClientEventIds)
      try LocalEventStore.shared.remove(clientEventIds: acknowledgedIds)
    }
  }
}
