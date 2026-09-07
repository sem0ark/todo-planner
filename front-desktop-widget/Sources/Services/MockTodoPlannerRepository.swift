import Foundation

final class MockTodoPlannerRepository: TodoPlannerRepository, @unchecked Sendable {
  private var mockToken: String? = "mock-jwt-token"
  private var categories: [Category] = []
  private var dayRecord: DayRecord

  init() {
    let now = Date()
    let initialCategories = Self.seedCategories()
    let initialPlan = Self.makeScheduleBlocks()
    self.categories = initialCategories
    self.dayRecord = DayRecord(
      calendarDate: Self.todayString(),
      plan: initialPlan,
      createdAt: now,
      updatedAt: now
    )
  }

  func getAuthToken() -> String? { mockToken }
  func persistAuthToken(_ token: String) async throws { mockToken = token }
  func clearAuth() async throws { mockToken = nil }
  func validateAuth() async throws -> Bool { mockToken != nil }

  func initialize(calendarDate: String) async throws -> InitResponse {
    if dayRecord.calendarDate != calendarDate {
      dayRecord = DayRecord(
        calendarDate: calendarDate,
        plan: Self.makeScheduleBlocks(),
        createdAt: Date(),
        updatedAt: Date()
      )
    }
    return InitResponse(
      settings: UserSettings(dayBoundaryTime: "04:00:00", updatedAt: Date()),
      categories: categories, dayRecord: dayRecord)
  }

  func submitEvents(calendarDate: String, events: [DayEvent]) async throws -> DayEventsResponse {
    for event in events {
      try LocalEventStore.shared.append(calendarDate: calendarDate, event: event)
    }
    let actual = events.filter { $0.eventType == "transition" }.map {
      ActualBlock(
        categoryId: $0.categoryId, blockType: "actual", startTime: timeString($0.occurredAt),
        durationMinutes: 0, isOpen: true)
    }
    dayRecord = DayRecord(
      calendarDate: calendarDate, plan: dayRecord.plan, actual: actual,
      createdAt: dayRecord.createdAt, updatedAt: Date())
    return DayEventsResponse(
      calendarDate: calendarDate, plan: dayRecord.plan, actual: actual,
      createdAt: dayRecord.createdAt, updatedAt: dayRecord.updatedAt)
  }

  func hasPendingSync() async -> Bool {
    (try? !LocalEventStore.shared.pendingEvents().isEmpty) ?? false
  }
  func synchronize() async throws {}

  private static func seedCategories() -> [Category] {
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

  private static func makeScheduleBlocks() -> [PlannedBlock] {
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
        categoryId: entry.1,
        startTime: entry.0,
        durationMinutes: minutesBetween(start: entry.0, end: nextStart))
    }
  }

  private static func minutesBetween(start: String, end: String) -> Int {
    let startParts = start.split(separator: ":").compactMap { Int($0) }
    let endParts = end.split(separator: ":").compactMap { Int($0) }
    guard startParts.count == 3, endParts.count == 3 else { return 0 }
    let startMinutes = startParts[0] * 60 + startParts[1]
    let endMinutes = endParts[0] * 60 + endParts[1]
    return max(1, endMinutes - startMinutes)
  }

  private static func todayString() -> String { DateFormatter.yyyyMMdd.string(from: Date()) }
  private func timeString(_ date: Date) -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "HH:mm:ss"
    return formatter.string(from: date)
  }
}
