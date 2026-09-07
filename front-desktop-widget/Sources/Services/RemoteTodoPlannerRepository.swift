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
    return await api.validateToken()
  }

  func initialize(calendarDate: String) async throws -> InitResponse {
    return try await api.initialize(calendarDate: calendarDate)
  }

  func submitEvents(calendarDate: String, events: [DayEvent]) async throws -> DayEventsResponse {
    return try await api.postDayEvents(date: calendarDate, events: events)
  }

  func hasPendingSync() async -> Bool { return false }  // Remote is always "synced"
  func synchronize() async throws {}
}
