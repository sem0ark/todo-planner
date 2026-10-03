import Foundation

struct DeviceRegistration: Codable {
  let deviceId: Int
  let registeredAt: Date
  enum CodingKeys: String, CodingKey {
    case deviceId = "device_id"
    case registeredAt = "registered_at"
  }
}

struct UserSettings: Codable {
  let dayRangeStartTime: String
  let dayRangeEndTime: String
  let updatedAt: Date
  enum CodingKeys: String, CodingKey {
    case dayRangeStartTime = "day_range_start_time"
    case dayRangeEndTime = "day_range_end_time"
    case updatedAt = "updated_at"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    dayRangeStartTime =
      try container.decodeIfPresent(String.self, forKey: .dayRangeStartTime)
      ?? "04:00:00"
    dayRangeEndTime =
      try container.decodeIfPresent(String.self, forKey: .dayRangeEndTime)
      ?? "23:00:00"
    updatedAt = try container.decode(Date.self, forKey: .updatedAt)
  }

  func encode(to encoder: Encoder) throws {
    var container = encoder.container(keyedBy: CodingKeys.self)
    try container.encode(dayRangeStartTime, forKey: .dayRangeStartTime)
    try container.encode(dayRangeEndTime, forKey: .dayRangeEndTime)
    try container.encode(updatedAt, forKey: .updatedAt)
  }

  init(
    dayRangeStartTime: String = "04:00:00",
    dayRangeEndTime: String = "23:00:00",
    updatedAt: Date
  ) {
    self.dayRangeStartTime = dayRangeStartTime
    self.dayRangeEndTime = dayRangeEndTime
    self.updatedAt = updatedAt
  }
}

struct InitResponse: Codable {
  let settings: UserSettings
  let categories: [Category]
  let dayRecord: DayRecord
  enum CodingKeys: String, CodingKey {
    case settings, categories
    case dayRecord = "day_record"
  }
}

struct CachedInitResponse: Codable {
  let schemaVersion: Int
  let calendarDate: String
  let fetchedAt: Date
  let response: InitResponse

  static let currentSchemaVersion = 1

  init(calendarDate: String, response: InitResponse, fetchedAt: Date = Date()) {
    self.schemaVersion = Self.currentSchemaVersion
    self.calendarDate = calendarDate
    self.fetchedAt = fetchedAt
    self.response = response
  }
}
