import Foundation

/// A high-fidelity mock repository for testing the Todo Planner widget.
/// Implements in-memory logic for transitions, offsets, and Pomodoro states.
class MockTodoPlannerRepository: TodoPlannerRepository, @unchecked Sendable {

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
    return nil
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
          categoryId: event.categoryId,
          blockType: "actual",
          startTime: timeStr,
          durationMinutes: 0  // Active block
        )
        updatedActuals.append(newBlock)

        createdEvents.append(
          CreatedEvent(
            id: eventId,
            eventType: event.eventType,
            categoryId: event.categoryId,
            occurredAt: event.occurredAt
          ))
      } else if event.eventType == "confirmation" {
        // Confirmations validate the current state
        print("[MOCK] Confirmation received for \(timeStr)")

        createdEvents.append(
          CreatedEvent(
            id: eventId,
            eventType: event.eventType,
            categoryId: event.categoryId,
            occurredAt: event.occurredAt
          ))
      }
    }

    // Update local state - create a new record with updated blocks
    let updatedRecord = DayRecord(
      id: currentRecord.id,
      calendarDate: currentRecord.calendarDate,
      actualBlocks: updatedActuals,
      createdAt: currentRecord.createdAt,
      updatedAt: Date()
    )
    self.dayRecord = updatedRecord

    return DayEventsResponse(createdEvents: createdEvents, actualBlocks: updatedActuals)
  }

  func fetchTodaySchedule() async throws -> TodaySchedule {
    let blocks = makeScheduleBlocks()
    let template = DayTemplate(
      id: 200,
      name: "Mock Daily Template",
      templateGroupId: nil,
      currentSnapshot: TemplateSnapshot(
        id: 201,
        snapshottedAt: Date(),
        snapshotBlocks: blocks
      )
    )
    return TodaySchedule(
      calendarDate: todayString(),
      dayTemplateId: 200,
      template: template
    )
  }

  func hasPendingSync() async -> Bool { return false }
  func synchronize() async throws {}

  // MARK: - Seeding Logic

  func seedCategories() -> [Category] {
    let now = Date()
    return [
      Category(
        id: 1, name: "Working", color: "#2563eb",
        pomodoroConfig: PomodoroConfig(workDuration: 45 * 60, restDuration: 5 * 60),
        createdAt: now, updatedAt: now),
      Category(
        id: 2, name: "Exercise", color: "#dc2626",
        pomodoroConfig: nil,
        createdAt: now, updatedAt: now),
      Category(
        id: 3, name: "Rest", color: "#0891b2",
        pomodoroConfig: nil,
        createdAt: now, updatedAt: now),
      Category(
        id: 4, name: "Learning", color: "#27b208",
        pomodoroConfig: PomodoroConfig(workDuration: 25 * 60, restDuration: 5 * 60),
        createdAt: now, updatedAt: now),
      Category(
        id: 5, name: "Housework", color: "#e9a663",
        pomodoroConfig: nil,
        createdAt: now, updatedAt: now),
    ]
  }

  func seedDayRecord(for date: String) -> DayRecord {
    return DayRecord(
      id: 5001,
      calendarDate: date,
      actualBlocks: [],  // Starts empty for testing initialization
      createdAt: Date(),
      updatedAt: Date()
    )
  }

  private func makeScheduleBlocks() -> [PlannedBlock] {
    let configuredEntries = BuildConfig.mockSchedule
      .split(separator: ",")
      .compactMap { entry -> (String, Int)? in
        let fields = entry.split(whereSeparator: { $0 == " " || $0 == "\t" })
        guard fields.count == 2, let categoryId = Int(fields[1]) else { return nil }
        return (String(fields[0]), categoryId)
      }

    let entries =
      configuredEntries.isEmpty
      ? (0..<24 * 60).map { minute in
        (String(format: "%02d:%02d:00", minute / 60, minute % 60), (minute % 5) + 1)
      }
      : configuredEntries

    return entries.enumerated().map { index, entry in
      let nextStart = index + 1 < entries.count ? entries[index + 1].0 : "24:00:00"
      return PlannedBlock(
        id: 101 + index,
        categoryId: entry.1,
        startTime: entry.0,
        durationMinutes: minutesBetween(start: entry.0, end: nextStart))
    }
  }

  private func minutesBetween(start: String, end: String) -> Int {
    let startParts = start.split(separator: ":").compactMap { Int($0) }
    let endParts = end.split(separator: ":").compactMap { Int($0) }
    guard startParts.count == 3, endParts.count == 3 else { return 0 }
    let startMinutes = startParts[0] * 60 + startParts[1]
    let endMinutes = endParts[0] * 60 + endParts[1]
    return max(1, endMinutes - startMinutes)
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
