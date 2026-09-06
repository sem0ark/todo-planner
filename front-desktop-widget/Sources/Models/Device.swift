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
  let dayBoundaryTime: String
  let updatedAt: Date
  enum CodingKeys: String, CodingKey {
    case dayBoundaryTime = "day_boundary_time"
    case updatedAt = "updated_at"
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
