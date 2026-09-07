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
    return try await api.postDayEvents(date: calendarDate, events: events)
  }

  func hasPendingSync() async -> Bool {
    // This repository has no local event queue, so there is nothing to sync.
    return false
  }

  func synchronize() async throws {
    WidgetLogger.error("Remote synchronization was requested but no local queue exists")
    throw StorageError.invalidContract("remote synchronization is not supported")
  }
}
