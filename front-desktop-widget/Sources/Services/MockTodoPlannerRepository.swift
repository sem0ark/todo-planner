import Foundation

/// A high-fidelity mock repository for testing the Todo Planner widget.
/// Implements in-memory logic for transitions, offsets, and Pomodoro states.
final class MockTodoPlannerRepository: TodoPlannerRepository, @unchecked Sendable {

  // MARK: - In-Memory Storage
  private var mockToken: String? = "mock-jwt-token"
  private var categories: [Category] = []
  private var dayRecord: DayRecord?

  // MARK: - Initializer
  init() {
    self.categories = seedCategories()
    self.dayRecord = seedDayRecord(for: todayString())
  }

  // MARK: - Authentication
  func getAuthToken() -> String? {
    return mockToken
  }

  func persistAuthToken(_ token: String) async throws {
    self.mockToken = token
  }

  func clearAuth() async throws {
    self.mockToken = nil
  }

  func validateAuth() async throws -> Bool {
    // Simulate network latency
    try await Task.sleep(nanoseconds: 100_000_000)
    return mockToken != nil
  }

  // MARK: - Categories
  func fetchCategories() async throws -> [Category] {
    return categories
  }

  // MARK: - Day Records
  func fetchDayRecord(date: String) async throws -> DayRecord? {
    if dayRecord?.calendarDate == date {
      return dayRecord
    }
    // If date doesn't match, create a new one for that date
    let newRecord = seedDayRecord(for: date)
    self.dayRecord = newRecord
    return newRecord
  }

  func createDayRecord(date: String) async throws -> DayRecord {
    let newRecord = seedDayRecord(for: date)
    self.dayRecord = newRecord
    return newRecord
  }

  // MARK: - Event Processing (The Logic Engine)
  func submitEvents(dayRecordId: Int, events: [DayEvent]) async throws -> DayEventsResponse {
    guard let currentRecord = self.dayRecord else {
      throw StorageError.notFound
    }

    var updatedActuals = currentRecord.actualBlocks
    var createdEvents: [CreatedEvent] = []

    for event in events {
      let timestamp = event.occurredAt
      let timeStr = formatTime(timestamp)
      let eventId = Int.random(in: 1000...9999)

      if event.eventType == "transition" {
        // 1. Close the previous block if it exists
        if let lastIndex = updatedActuals.indices.last {
          let lastBlock = updatedActuals[lastIndex]
          let duration = calculateMinutesBetween(start: lastBlock.startTime, end: timeStr)
          updatedActuals[lastIndex] = ActualBlock(
            id: lastBlock.id,
            categoryId: lastBlock.categoryId,
            blockType: lastBlock.blockType,
            startTime: lastBlock.startTime,
            durationMinutes: max(1, duration)
          )
        }

        // 2. Start the new block
        let newBlock = ActualBlock(
          id: Int.random(in: 1000...9999),
          categoryId: event.incomingCategoryId,
          blockType: "actual",
          startTime: timeStr,
          durationMinutes: 0  // Active block
        )
        updatedActuals.append(newBlock)

        createdEvents.append(
          CreatedEvent(
            id: eventId,
            eventType: event.eventType,
            outgoingCategoryId: event.outgoingCategoryId,
            incomingCategoryId: event.incomingCategoryId,
            occurredAt: event.occurredAt
          ))
      } else if event.eventType == "confirmation" {
        // Confirmations validate the current state
        print("[MOCK] Confirmation received for \(timeStr)")

        createdEvents.append(
          CreatedEvent(
            id: eventId,
            eventType: event.eventType,
            outgoingCategoryId: event.outgoingCategoryId,
            incomingCategoryId: event.incomingCategoryId,
            occurredAt: event.occurredAt
          ))
      }
    }

    // Update local state - create a new record with updated blocks
    let updatedRecord = DayRecord(
      id: currentRecord.id,
      snapshotId: currentRecord.snapshotId,
      calendarDate: currentRecord.calendarDate,
      reviewStatus: currentRecord.reviewStatus,
      snapshotBlocks: currentRecord.snapshotBlocks,
      actualBlocks: updatedActuals,
      createdAt: currentRecord.createdAt,
      updatedAt: Date()
    )
    self.dayRecord = updatedRecord

    return DayEventsResponse(createdEvents: createdEvents, actualBlocks: updatedActuals)
  }

  func hasPendingSync() async -> Bool { return false }
  func synchronize() async throws {}

  // MARK: - Private Seeding Logic

  private func seedCategories() -> [Category] {
    let now = Date()
    return [
      Category(
        id: 1, name: "Deep Work", color: "#1e40af",  // Blue
        pomodoroConfig: PomodoroConfig(workDuration: 1500, restDuration: 300),
        createdAt: now, updatedAt: now),
      Category(
        id: 2, name: "Admin", color: "#78716c",  // Stone
        pomodoroConfig: nil,
        createdAt: now, updatedAt: now),
      Category(
        id: 3, name: "Communication", color: "#7c3aed",  // Violet
        pomodoroConfig: nil,
        createdAt: now, updatedAt: now),
      Category(
        id: 4, name: "Learning", color: "#0891b2",  // Cyan
        pomodoroConfig: PomodoroConfig(workDuration: 1800, restDuration: 300),
        createdAt: now, updatedAt: now),
      Category(
        id: 5, name: "Health", color: "#16a34a",  // Green
        pomodoroConfig: nil,
        createdAt: now, updatedAt: now),
      Category(
        id: 6, name: "Break", color: "#94a3b8",  // Slate
        pomodoroConfig: nil,
        createdAt: now, updatedAt: now),
    ]
  }

  private func seedDayRecord(for date: String) -> DayRecord {
    // Create a mock "Ideal Day" template with a realistic schedule
    let snapshot = [
      PlannedBlock(id: 101, categoryId: 1, startTime: "09:00:00", durationMinutes: 90),  // Deep Work
      PlannedBlock(id: 102, categoryId: 6, startTime: "10:30:00", durationMinutes: 15),  // Break
      PlannedBlock(id: 103, categoryId: 1, startTime: "10:45:00", durationMinutes: 90),  // Deep Work
      PlannedBlock(id: 104, categoryId: 6, startTime: "12:15:00", durationMinutes: 45),  // Lunch
      PlannedBlock(id: 105, categoryId: 3, startTime: "13:00:00", durationMinutes: 60),  // Communication
      PlannedBlock(id: 106, categoryId: 2, startTime: "14:00:00", durationMinutes: 60),  // Admin
      PlannedBlock(id: 107, categoryId: 4, startTime: "15:00:00", durationMinutes: 90),  // Learning
      PlannedBlock(id: 108, categoryId: 5, startTime: "16:30:00", durationMinutes: 60),  // Health
    ]

    return DayRecord(
      id: 5001,
      snapshotId: 200,
      calendarDate: date,
      reviewStatus: "unreviewed",
      snapshotBlocks: snapshot,
      actualBlocks: [],  // Starts empty for testing initialization
      createdAt: Date(),
      updatedAt: Date()
    )
  }

  // MARK: - Helpers

  private func todayString() -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    return formatter.string(from: Date())
  }

  private func formatTime(_ date: Date) -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "HH:mm:ss"
    return formatter.string(from: date)
  }

  private func calculateMinutesBetween(start: String, end: String) -> Int {
    let formatter = DateFormatter()
    formatter.dateFormat = "HH:mm:ss"
    guard let startDate = formatter.date(from: start),
      let endDate = formatter.date(from: end)
    else { return 0 }
    return Int(endDate.timeIntervalSince(startDate) / 60)
  }
}
