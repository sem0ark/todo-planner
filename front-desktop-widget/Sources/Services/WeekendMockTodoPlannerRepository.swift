import Foundation

final class WeekendMockTodoPlannerRepository: MockTodoPlannerRepository, @unchecked Sendable {
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
        pomodoroConfig: PomodoroConfig(workDuration: 25 * 60, restDuration: 5 * 60),
        createdAt: now,
        updatedAt: now
      ),
      Category(
        id: 5,
        name: "Housework",
        color: "#e9a663",
        pomodoroConfig: nil,
        createdAt: now,
        updatedAt: now
      ),
    ]
  }

  override func seedDayRecord(for date: String) -> DayRecord {
    let schedule: [(String, Int, Int)] = [
      ("04:00:00", 3, 420),
      ("11:00:00", 4, 120),
      ("13:00:00", 3, 60),
      ("14:00:00", 1, 180),
      ("17:00:00", 2, 60),
      ("18:00:00", 4, 150),
      ("20:30:00", 3, 450),
    ]

    let blocks = schedule.enumerated().map { index, item in
      PlannedBlock(
        id: 400 + index,
        categoryId: item.1,
        startTime: item.0,
        durationMinutes: item.2
      )
    }

    let now = Date()
    return DayRecord(
      id: 6002,
      snapshotId: 400,
      calendarDate: date,
      reviewStatus: "unreviewed",
      snapshotBlocks: blocks,
      actualBlocks: [],
      createdAt: now,
      updatedAt: now
    )
  }
}