import Foundation

struct PlannedBlock: Codable, Identifiable {
  let categoryId: Int
  let startTime: String
  let durationMinutes: Int

  var id: String { "\(categoryId)-\(startTime)-\(durationMinutes)" }

  enum CodingKeys: String, CodingKey {
    case categoryId = "category_id"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
  }
}

struct ActualBlock: Codable, Identifiable {
  let categoryId: Int?
  let blockType: String
  let startTime: String
  let durationMinutes: Int
  let isOpen: Bool

  var id: String { "\(categoryId.map(String.init) ?? "none")-\(startTime)-\(blockType)" }

  enum CodingKeys: String, CodingKey {
    case categoryId = "category_id"
    case blockType = "block_type"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
    case isOpen = "is_open"
  }

  init(
    categoryId: Int?, blockType: String, startTime: String,
    durationMinutes: Int, isOpen: Bool = false
  ) {
    self.categoryId = categoryId
    self.blockType = blockType
    self.startTime = startTime
    self.durationMinutes = durationMinutes
    self.isOpen = isOpen
  }
}

struct DayRecord: Codable {
  let calendarDate: String
  let dayTemplateId: Int?
  let plan: [PlannedBlock]
  let actual: [ActualBlock]
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case calendarDate = "calendar_date"
    case dayTemplateId = "day_template_id"
    case plan, actual
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }

  init(
    calendarDate: String, dayTemplateId: Int? = nil,
    plan: [PlannedBlock] = [], actual: [ActualBlock] = [],
    createdAt: Date, updatedAt: Date
  ) {
    self.calendarDate = calendarDate
    self.dayTemplateId = dayTemplateId
    self.plan = plan
    self.actual = actual
    self.createdAt = createdAt
    self.updatedAt = updatedAt
  }
}

struct DayRecordsResponse: Decodable {
  let dayRecords: [DayRecord]

  enum CodingKeys: String, CodingKey {
    case dayRecords = "day_records"
    case days
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    if let records = try container.decodeIfPresent([DayRecord].self, forKey: .dayRecords) {
      dayRecords = records
    } else {
      let entries = try container.decodeIfPresent([DayRangeEntry].self, forKey: .days) ?? []
      dayRecords = entries.compactMap(\.dayRecord)
    }
  }
}

private struct DayRangeEntry: Decodable {
  let dayRecord: DayRecord?
  enum CodingKeys: String, CodingKey { case dayRecord = "day_record" }
}
