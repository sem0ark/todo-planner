import Foundation

struct PendingDayEvent: Codable, Sendable {
  let calendarDate: String
  let event: DayEvent
}

typealias LocalEventStoreFileWriter = (Data, URL) throws -> Void

/// Durable storage for the pending queue and the complete local event backup.
final class LocalEventStore: @unchecked Sendable {
  static let shared = LocalEventStore()

  private let fileURL: URL
  private let backupFileURL: URL
  private let legacyFileURL: URL
  private let legacyBackupFileURL: URL
  private let lock = NSLock()
  private let appendFile: LocalEventStoreFileWriter
  private let atomicWriteFile: LocalEventStoreFileWriter

  init(
    fileManager: FileManager = .default,
    storageDirectory: URL? = nil,
    appendFile: @escaping LocalEventStoreFileWriter = LocalEventStore.defaultAppendFile,
    atomicWriteFile: @escaping LocalEventStoreFileWriter = LocalEventStore.defaultAtomicWriteFile
  ) {
    let directory: URL
    if let storageDirectory {
      directory = storageDirectory
    } else {
      let applicationSupport = fileManager.urls(
        for: .applicationSupportDirectory, in: .userDomainMask)[0]
      directory = applicationSupport.appendingPathComponent(
        "TodoPlannerWidget-\(Self.storageNamespace)", isDirectory: true)
    }
    try? fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
    fileURL = directory.appendingPathComponent("pending-events.jsonl")
    backupFileURL = directory.appendingPathComponent("backup-events.jsonl")
    legacyFileURL = directory.appendingPathComponent("pending-events.json")
    legacyBackupFileURL = directory.appendingPathComponent("backup-events.json")
    self.appendFile = appendFile
    self.atomicWriteFile = atomicWriteFile
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
    let pendingEvents = try loadUnlocked()
    let entry = PendingDayEvent(calendarDate: calendarDate, event: event)
    var backupEvents = try loadBackupUnlocked()
    if !FileManager.default.fileExists(atPath: fileURL.path) {
      try saveJSONLinesUnlocked(pendingEvents, to: fileURL)
    }
    if !FileManager.default.fileExists(atPath: backupFileURL.path) {
      try saveJSONLinesUnlocked(backupEvents, to: backupFileURL)
    }
    if !backupEvents.contains(where: { $0.event.clientEventId == event.clientEventId }) {
      try appendUnlocked(entry, to: backupFileURL)
    }
    try appendUnlocked(entry, to: fileURL)
  }

  func pendingEvents() throws -> [PendingDayEvent] {
    lock.lock()
    defer { lock.unlock() }
    return try loadUnlocked()
  }

  func backupEvents() throws -> [PendingDayEvent] {
    lock.lock()
    defer { lock.unlock() }
    return try loadBackupUnlocked()
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
    try loadJSONLinesUnlocked(fileURL: fileURL, legacyFileURL: legacyFileURL)
  }

  private func saveUnlocked(_ pendingEvents: [PendingDayEvent]) throws {
    try saveJSONLinesUnlocked(pendingEvents, to: fileURL)
  }

  private func loadBackupUnlocked() throws -> [PendingDayEvent] {
    try loadJSONLinesUnlocked(fileURL: backupFileURL, legacyFileURL: legacyBackupFileURL)
  }

  private func appendUnlocked(_ event: PendingDayEvent, to fileURL: URL) throws {
    let lineData = try JSONEncoder.widgetEncoder.encode(event) + Data([0x0A])
    try appendFile(lineData, fileURL)
  }

  private func loadJSONLinesUnlocked(fileURL: URL, legacyFileURL: URL) throws -> [PendingDayEvent] {
    guard FileManager.default.fileExists(atPath: fileURL.path) else {
      return try loadLegacyUnlocked(from: legacyFileURL)
    }

    let data = try Data(contentsOf: fileURL)
    do {
      return try decodeJSONLines(data)
    } catch {
      guard FileManager.default.fileExists(atPath: legacyFileURL.path) else { throw error }
      let legacyEvents = try loadLegacyUnlocked(from: legacyFileURL)
      try saveJSONLinesUnlocked(legacyEvents, to: fileURL)
      return legacyEvents
    }
  }

  private func loadLegacyUnlocked(from fileURL: URL) throws -> [PendingDayEvent] {
    guard FileManager.default.fileExists(atPath: fileURL.path) else { return [] }
    let data = try Data(contentsOf: fileURL)
    return try JSONDecoder.widgetDecoder.decode([PendingDayEvent].self, from: data)
  }

  private func decodeJSONLines(_ data: Data) throws -> [PendingDayEvent] {
    guard let contents = String(data: data, encoding: .utf8) else {
      throw StorageError.decodingError
    }
    return
      try contents
      .split(whereSeparator: \.isNewline)
      .map { try JSONDecoder.widgetDecoder.decode(PendingDayEvent.self, from: Data($0.utf8)) }
  }

  private func saveJSONLinesUnlocked(_ events: [PendingDayEvent], to fileURL: URL) throws {
    let data = try events.reduce(into: Data()) { result, event in
      result.append(try JSONEncoder.widgetEncoder.encode(event))
      result.append(0x0A)
    }
    try atomicWriteFile(data, fileURL)
  }

  private static func defaultAppendFile(data: Data, fileURL: URL) throws {
    if FileManager.default.fileExists(atPath: fileURL.path) {
      let existingData = try Data(contentsOf: fileURL)
      try (existingData + data).write(to: fileURL, options: .atomic)
    } else {
      try data.write(to: fileURL, options: .atomic)
    }
  }

  private static func defaultAtomicWriteFile(data: Data, fileURL: URL) throws {
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
