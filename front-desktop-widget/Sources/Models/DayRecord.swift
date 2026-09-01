import Foundation

struct PlannedBlock: Identifiable, Codable {
  let id: Int
  let categoryId: Int
  let startTime: String  // HH:MM:SS format
  let durationMinutes: Int

  enum CodingKeys: String, CodingKey {
    case id
    case categoryId = "category_id"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
  }
}

struct ActualBlock: Identifiable, Codable {
  let id: Int
  let categoryId: Int?
  let blockType: String  // "actual" | "blank" | "untracked"
  let startTime: String  // HH:MM:SS format
  let durationMinutes: Int

  enum CodingKeys: String, CodingKey {
    case id
    case categoryId = "category_id"
    case blockType = "block_type"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
  }
}

struct DayRecord: Identifiable, Codable {
  let id: Int
  let calendarDate: String  // YYYY-MM-DD
  let reviewStatus: String  // "Unreviewed" | "Reviewed" | "Ignored"
  let actualBlocks: [ActualBlock]
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case calendarDate = "calendar_date"
    case reviewStatus = "review_status"
    case actualBlocks = "actual_blocks"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }

  init(
    id: Int, calendarDate: String, reviewStatus: String,
    actualBlocks: [ActualBlock], createdAt: Date, updatedAt: Date
  ) {
    self.id = id
    self.calendarDate = calendarDate
    self.reviewStatus = reviewStatus
    self.actualBlocks = actualBlocks
    self.createdAt = createdAt
    self.updatedAt = updatedAt
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    id = try container.decode(Int.self, forKey: .id)
    calendarDate = try container.decode(String.self, forKey: .calendarDate)
    reviewStatus = try container.decode(String.self, forKey: .reviewStatus)
    actualBlocks = try container.decodeIfPresent([ActualBlock].self, forKey: .actualBlocks) ?? []
    createdAt = try container.decode(Date.self, forKey: .createdAt)
    updatedAt = try container.decode(Date.self, forKey: .updatedAt)
  }
}

struct DayRecordsResponse: Codable {
  let dayRecords: [DayRecord]

  enum CodingKeys: String, CodingKey {
    case dayRecords = "day_records"
  }
}
