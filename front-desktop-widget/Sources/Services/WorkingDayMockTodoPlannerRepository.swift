import Foundation

final class WorkingDayMockTodoPlannerRepository: MockTodoPlannerRepository, @unchecked Sendable {
  override func seedCategories() -> [Category] {
    let now = Date()
    return [
      Category(
        id: 1,
        name: "Working",
        color: "#2563eb",
        pomodoroConfig: PomodoroConfig(workDuration: 45 * 60, restDuration: 5 * 60),
        createdAt: now,
        updatedAt: now
      ),
      Category(
        id: 2,
        name: "Exercise",
        color: "#dc2626",
        pomodoroConfig: nil,
        createdAt: now,
        updatedAt: now
      ),
      Category(
        id: 3,
        name: "Rest",
        color: "#0891b2",
        pomodoroConfig: nil,
        createdAt: now,
        updatedAt: now
      ),
      Category(
        id: 4,
        name: "Learning",
        color: "#27b208",
        pomodoroConfig: nil,
        createdAt: now,
        updatedAt: now
      ),
    ]
  }

  override func seedDayRecord(for date: String) -> DayRecord {
    let schedule: [(String, Int, Int)] = [
      ("00:00:00", 3, 480),  // Rest
      ("08:00:00", 4, 60),  // Learning
      ("09:00:00", 1, 180),  // Working
      ("12:00:00", 3, 60),  // Lunch (Rest)
      ("13:00:00", 1, 300),  // Working
      ("18:00:00", 2, 60),  // Exercise
      ("19:00:00", 3, 360),  // Rest
    ]

    let blocks = schedule.enumerated().map { index, item in
      PlannedBlock(
        id: 300 + index,
        categoryId: item.1,
        startTime: item.0,
        durationMinutes: item.2
      )
    }

    let now = Date()
    return DayRecord(
      id: 6001,
      snapshotId: 300,
      calendarDate: date,
      reviewStatus: "unreviewed",
      snapshotBlocks: blocks,
      actualBlocks: [],
      createdAt: now,
      updatedAt: now
    )
  }
}
