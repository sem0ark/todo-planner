import Foundation

struct PendingDayEvent: Codable, Sendable {
  let calendarDate: String
  let event: DayEvent
}

/// Durable append-only storage for events waiting for the next sync.
final class LocalEventStore: @unchecked Sendable {
  static let shared = LocalEventStore()

  private let fileURL: URL
  private let lock = NSLock()

  init(fileManager: FileManager = .default) {
    let applicationSupport = fileManager.urls(
      for: .applicationSupportDirectory, in: .userDomainMask)[0]
    let directory = applicationSupport.appendingPathComponent(
      "TodoPlannerWidget-\(Self.storageNamespace)", isDirectory: true)
    try? fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
    fileURL = directory.appendingPathComponent("pending-events.json")
  }

  private static var storageNamespace: String {
    if ProcessInfo.processInfo.processName == "test_runner" {
      return "tests"
    }
    return BuildConfig.storageMode == "mock" ? "mock" : "remote"
  }

  func append(calendarDate: String, event: DayEvent) throws {
    lock.lock()
    defer { lock.unlock() }
    var pendingEvents = try loadUnlocked()
    pendingEvents.append(PendingDayEvent(calendarDate: calendarDate, event: event))
    try saveUnlocked(pendingEvents)
  }

  func pendingEvents() throws -> [PendingDayEvent] {
    lock.lock()
    defer { lock.unlock() }
    return try loadUnlocked()
  }

  func remove(clientEventIds: Set<String>) throws {
    guard !clientEventIds.isEmpty else { return }
    lock.lock()
    defer { lock.unlock() }
    let remainingEvents = try loadUnlocked().filter {
      !clientEventIds.contains($0.event.clientEventId)
    }
    try saveUnlocked(remainingEvents)
  }

  private func loadUnlocked() throws -> [PendingDayEvent] {
    guard FileManager.default.fileExists(atPath: fileURL.path) else { return [] }
    let data = try Data(contentsOf: fileURL)
    return try JSONDecoder.widgetDecoder.decode([PendingDayEvent].self, from: data)
  }

  private func saveUnlocked(_ pendingEvents: [PendingDayEvent]) throws {
    let data = try JSONEncoder.widgetEncoder.encode(pendingEvents)
    try data.write(to: fileURL, options: .atomic)
  }
}

extension JSONEncoder {
  static var widgetEncoder: JSONEncoder {
    let encoder = JSONEncoder()
    encoder.dateEncodingStrategy = .iso8601
    return encoder
  }
}

extension JSONDecoder {
  static var widgetDecoder: JSONDecoder {
    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    return decoder
  }
}
