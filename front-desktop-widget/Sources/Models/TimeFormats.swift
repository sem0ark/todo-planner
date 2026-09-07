import Foundation

enum TimeFormats {
  static let calendarDate = "yyyy-MM-dd"
  static let scheduleTime = "HH:mm:ss"
  static let timestamp = "yyyy-MM-dd'T'HH:mm:ssXXXXX"

  static func parseScheduleTime(_ value: String) -> Date? {
    guard value.count == scheduleTime.count else { return nil }
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.dateFormat = scheduleTime
    return formatter.date(from: value)
  }

  static func secondsSinceDayStart(_ value: String) -> Int? {
    guard let date = parseScheduleTime(value) else { return nil }
    let components = Calendar(identifier: .gregorian).dateComponents(
      [.hour, .minute, .second], from: date)
    guard let hour = components.hour, let minute = components.minute, let second = components.second
    else { return nil }
    return hour * 3600 + minute * 60 + second
  }
}
