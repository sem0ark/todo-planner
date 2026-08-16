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
    ]
  }

  override func seedDayRecord(for date: String) -> DayRecord {
    // Schedule breakdown:
    // 00:00 - 08:00: Rest (480 min)
    // 08:00 - 12:00: Working (240 min)
    // 12:00 - 13:00: Rest/Lunch (60 min)
    // 13:00 - 17:00: Working (240 min)
    // 17:00 - 18:00: Exercise (60 min)
    // 18:00 - 00:00: Rest (360 min)

    let schedule: [(String, Int, Int)] = [
      ("00:00:00", 3, 480),  // Rest
      ("08:00:00", 1, 240),  // Working
      ("12:00:00", 3, 60),  // Lunch (Rest)
      ("13:00:00", 1, 240),  // Working
      ("17:00:00", 2, 60),  // Exercise
      ("18:00:00", 3, 360),  // Rest
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
