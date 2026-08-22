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

  func fetchCategories() async throws -> [Category] {
    return try await api.fetchCategories()
  }

  func fetchDayRecord(date: String) async throws -> DayRecord? {
    let records = try await api.fetchDayRecords(from: date, to: date)
    return records.first
  }

  func createDayRecord(date: String) async throws -> DayRecord {
    return try await api.createDayRecord(date: date)
  }

  func submitEvents(dayRecordId: Int, events: [DayEvent]) async throws -> DayEventsResponse {
    return try await api.postDayEvents(dayRecordId: dayRecordId, events: events)
  }

  func hasPendingSync() async -> Bool { return false }  // Remote is always "synced"
  func synchronize() async throws {}
}
