import Foundation

enum TimeFormats {
  static let calendarDate = "yyyy-MM-dd"
  static let scheduleTime = "HH:mm:ss"
  static let timestamp = "yyyy-MM-dd'T'HH:mm:ssXXXXX"
  static let localTimestamp = "yyyy-MM-dd'T'HH:mm:ssxxx"

  static func localTimestamp(for date: Date) -> String {
    return timestamp(for: date, timeZone: .current)
  }

  static func localTimestamp(for date: Date, preservingOffsetFrom timestamp: String) -> String? {
    let offset = String(timestamp.suffix(6))
    guard (offset.first == "+" || offset.first == "-"), offset.dropFirst().contains(":") else {
      return nil
    }
    guard let timeZone = timeZone(for: offset) else { return nil }
    return self.timestamp(for: date, timeZone: timeZone)
  }

  private static func timestamp(for date: Date, timeZone: TimeZone) -> String {
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.calendar = Calendar(identifier: .gregorian)
    formatter.timeZone = timeZone
    formatter.dateFormat = localTimestamp
    return formatter.string(from: date)
  }

  private static func timeZone(for offset: String) -> TimeZone? {
    let sign = offset.first == "+" ? 1 : -1
    let components = offset.dropFirst().split(separator: ":")
    guard components.count == 2,
      let hours = Int(components[0]), let minutes = Int(components[1]) else { return nil }
    return TimeZone(secondsFromGMT: sign * (hours * 60 + minutes) * 60)
  }

  static func parseScheduleTime(_ value: String) -> Date? {
    guard value.count == scheduleTime.count else {
      WidgetLogger.error(
        "Invalid schedule time", context: ["value": value, "expected": scheduleTime])
      return nil
    }
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.dateFormat = scheduleTime
    guard let date = formatter.date(from: value) else {
      WidgetLogger.error(
        "Invalid schedule time", context: ["value": value, "expected": scheduleTime])
      return nil
    }
    return date
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
