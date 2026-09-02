import Foundation

/// Errors specific to the storage and retrieval layer.
enum StorageError: Error {
  case unauthorized
  case networkFailure(Error)
  case databaseError(String)
  case notFound
  case decodingError
}

/// The universal interface for data operations.
protocol TodoPlannerRepository: Sendable {
  // MARK: - Authentication
  func getAuthToken() -> String?
  func persistAuthToken(_ token: String) async throws
  func clearAuth() async throws
  func validateAuth() async throws -> Bool

  // MARK: - Categories
  func fetchCategories() async throws -> [Category]

  // MARK: - Day Records
  func fetchDayRecord(date: String) async throws -> DayRecord?
  func createDayRecord(date: String) async throws -> DayRecord

  // MARK: - Schedule
  func fetchTodaySchedule() async throws -> TodaySchedule

  // MARK: - Events & Reality Logging
  /// Submits events and returns the updated state of actual blocks.
  func submitEvents(dayRecordId: Int, events: [DayEvent]) async throws -> DayEventsResponse

  // MARK: - Sync & Persistence (For SQLite/Offline)
  func hasPendingSync() async -> Bool
  func synchronize() async throws
}
