import Foundation

struct TodaySchedule: Codable {
  let calendarDate: String
  let dayTemplateId: Int?
  let template: DayTemplate?

  enum CodingKeys: String, CodingKey {
    case calendarDate = "calendar_date"
    case dayTemplateId = "day_template_id"
    case template
  }
}

struct DayTemplate: Codable {
  let id: Int
  let name: String
  let templateGroupId: Int?
  let currentSnapshot: TemplateSnapshot

  enum CodingKeys: String, CodingKey {
    case id
    case name
    case templateGroupId = "template_group_id"
    case currentSnapshot = "current_snapshot"
  }
}

struct TemplateRequest: Encodable {
  let name: String
  let templateGroupId: Int?
  let snapshotBlocks: [TemplateBlockRequest]

  enum CodingKeys: String, CodingKey {
    case name
    case templateGroupId = "template_group_id"
    case snapshotBlocks = "snapshot_blocks"
  }
}

struct TemplateBlockRequest: Encodable {
  let categoryId: Int
  let startTime: String
  let durationMinutes: Int

  enum CodingKeys: String, CodingKey {
    case categoryId = "category_id"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
  }
}

struct TemplateSnapshot: Codable {
  let id: Int
  let snapshottedAt: Date
  let snapshotBlocks: [PlannedBlock]

  enum CodingKeys: String, CodingKey {
    case id
    case snapshottedAt = "snapshotted_at"
    case snapshotBlocks = "snapshot_blocks"
  }
}
