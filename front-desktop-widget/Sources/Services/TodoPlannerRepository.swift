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

  // MARK: - Client Bootstrap
  func initialize(calendarDate: String) async throws -> InitResponse

  // MARK: - Events & Reality Logging
  /// Submits events for a calendar date and returns derived actual blocks.
  func submitEvents(calendarDate: String, events: [DayEvent]) async throws -> DayEventsResponse

  // MARK: - Sync & Persistence (For SQLite/Offline)
  func hasPendingSync() async -> Bool
  func synchronize() async throws
}
