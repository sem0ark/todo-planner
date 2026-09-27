import Foundation

struct PlannedBlock: Codable, Identifiable {
  let categoryId: Int
  let startTime: String
  let durationMinutes: Int

  var id: String { "\(categoryId)-\(startTime)-\(durationMinutes)" }
  var startSeconds: Int? { TimeFormats.secondsSinceDayStart(startTime) }
  var durationSeconds: Int { durationMinutes * 60 }

  enum CodingKeys: String, CodingKey {
    case categoryId = "category_id"
    case startTime = "start_time"
    case durationMinutes = "duration_minutes"
  }

}

struct DayRecord: Codable {
  let calendarDate: String
  let plan: [PlannedBlock]

  enum CodingKeys: String, CodingKey {
    case calendarDate = "calendar_date"
    case plan
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    calendarDate = try container.decode(String.self, forKey: .calendarDate)
    plan = try container.decode([PlannedBlock].self, forKey: .plan)
  }

  init(
    calendarDate: String, plan: [PlannedBlock] = []
  ) {
    self.calendarDate = calendarDate
    self.plan = plan
  }
}

struct DayRecordsResponse: Decodable {
  let dayRecords: [DayRecord]

  enum CodingKeys: String, CodingKey {
    case dayRecords = "day_records"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    if let records = try container.decodeIfPresent([DayRecord].self, forKey: .dayRecords) {
      dayRecords = records
      return
    }
    throw DecodingError.keyNotFound(
      CodingKeys.dayRecords,
      DecodingError.Context(
        codingPath: decoder.codingPath,
        debugDescription: "Response must contain day_records"))
  }
}
