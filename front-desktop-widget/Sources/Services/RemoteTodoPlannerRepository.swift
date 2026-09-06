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
    let response = try await api.initialize(calendarDate: todayString())
    return response.categories
  }

  func fetchDayRecord(date: String) async throws -> DayRecord? {
    return try await api.initialize(calendarDate: date).dayRecord
  }

  func createDayRecord(date: String) async throws -> DayRecord {
    return try await api.initialize(calendarDate: date).dayRecord
  }

  func submitEvents(dayRecordId: Int, events: [DayEvent]) async throws -> DayEventsResponse {
    return try await api.postCurrentDayEvents(events: events)
  }

  func fetchTodaySchedule() async throws -> TodaySchedule {
    let day = try await api.initialize(calendarDate: todayString()).dayRecord
    let template: DayTemplate? = day.dayTemplateId.map { templateId in
      DayTemplate(
        id: templateId,
        name: "",
        templateGroupId: nil,
        currentSnapshot: TemplateSnapshot(
          id: day.snapshotId ?? 0,
          snapshottedAt: day.updatedAt,
          snapshotBlocks: day.snapshotBlocks
        )
      )
    }
    return TodaySchedule(
      calendarDate: day.calendarDate,
      dayTemplateId: day.dayTemplateId,
      template: template
    )
  }

  private func todayString() -> String {
    DateFormatter.yyyyMMdd.string(from: Date())
  }

  func hasPendingSync() async -> Bool { return false }  // Remote is always "synced"
  func synchronize() async throws {}
}
