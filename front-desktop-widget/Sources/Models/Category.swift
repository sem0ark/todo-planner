import Foundation

struct PomodoroConfig: Codable, Equatable {
  let workDuration: Int  // in seconds
  let restDuration: Int  // in seconds

  enum CodingKeys: String, CodingKey {
    case workDuration = "work_duration"
    case restDuration = "rest_duration"
  }
}

struct Category: Identifiable, Codable {
  let id: Int
  let name: String
  let color: String  // Hex color
  let pomodoroConfig: PomodoroConfig?
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case name
    case color
    case pomodoroConfig = "pomodoro_config"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }

  var hasPomodoroEnabled: Bool {
    return pomodoroConfig != nil
  }
}

struct CategoriesResponse: Codable {
  let categories: [Category]
}
