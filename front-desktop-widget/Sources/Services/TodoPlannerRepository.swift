import Foundation
import os

enum WidgetLogger {
  private static let logger = Logger(
    subsystem: "com.todoplanner.widget",
    category: "widget"
  )

  static func format(_ message: String, context: [String: String] = [:]) -> String {
    let details = context.sorted { first, second in first.key < second.key }
      .map { entry in "\(entry.key)=\(entry.value)" }.joined(separator: " ")
    return "\(message)\(details.isEmpty ? "" : " | \(details)")"
  }

  static func debug(_ message: String, context: [String: String] = [:]) {
    let formattedMessage = format("[DEBUG] \(message)", context: context)
    print(formattedMessage)
    logger.debug("\(formattedMessage, privacy: .public)")
  }

  static func error(_ message: String, context: [String: String] = [:]) {
    let formattedMessage = format("[ERROR] \(message)", context: context)
    print(formattedMessage)
    logger.error("\(formattedMessage, privacy: .public)")
  }
}

/// Errors specific to the storage and retrieval layer.
enum StorageError: Error {
  case unauthorized
  case networkFailure(Error)
  case databaseError(String)
  case notFound
  case decodingError
  case invalidContract(String)
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
  func cachedInitialization(calendarDate: String) throws -> InitResponse?

  // MARK: - Events & Reality Logging
  /// Appends events to the local log; network synchronization happens separately.
  func submitEvents(calendarDate: String, events: [DayEvent]) async throws -> DayEventsResponse

  // MARK: - Sync & Persistence (For SQLite/Offline)
  func hasPendingSync() async -> Bool
  func synchronize() async throws
  func clearLocalData() throws
}
