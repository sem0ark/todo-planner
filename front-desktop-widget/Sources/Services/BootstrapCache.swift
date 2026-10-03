import Foundation

final class BootstrapCache: @unchecked Sendable {
  static let shared = BootstrapCache()

  private let directory: URL
  private let lock = NSLock()

  init(fileManager: FileManager = .default, storageDirectory: URL? = nil) {
    if let storageDirectory {
      directory = storageDirectory
    } else {
      let applicationSupport = fileManager.urls(
        for: .applicationSupportDirectory, in: .userDomainMask)[0]
      let namespace = BuildConfig.storageMode == "mock" ? "mock" : "remote"
      directory =
        applicationSupport
        .appendingPathComponent("TodoPlannerWidget-\(namespace)", isDirectory: true)
        .appendingPathComponent("bootstrap-cache", isDirectory: true)
    }

    try? fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
  }

  func load(calendarDate: String) throws -> CachedInitResponse? {
    lock.lock()
    defer { lock.unlock() }

    let fileURL = cacheURL(calendarDate: calendarDate)
    guard FileManager.default.fileExists(atPath: fileURL.path) else {
      WidgetLogger.debug("Bootstrap cache miss", context: ["calendarDate": calendarDate])
      return nil
    }

    do {
      let data = try Data(contentsOf: fileURL)
      let cachedResponse = try JSONDecoder.widgetDecoder.decode(CachedInitResponse.self, from: data)
      guard cachedResponse.schemaVersion == CachedInitResponse.currentSchemaVersion,
        cachedResponse.calendarDate == calendarDate,
        cachedResponse.dayRecordsContainOnlyCalendarDate
      else {
        WidgetLogger.debug(
          "Bootstrap cache rejected", context: ["calendarDate": calendarDate])
        return nil
      }
      WidgetLogger.debug("Bootstrap cache hit", context: ["calendarDate": calendarDate])
      return cachedResponse
    } catch {
      WidgetLogger.error(
        "Unable to load bootstrap cache",
        context: ["calendarDate": calendarDate, "error": String(describing: error)])
      return nil
    }
  }

  func save(response: InitResponse, calendarDate: String) throws {
    guard response.dayRecords.contains(where: { $0.calendarDate == calendarDate }) else {
      throw StorageError.invalidContract("bootstrap calendar date does not match request")
    }

    lock.lock()
    defer { lock.unlock() }

    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    for dayRecord in response.dayRecords {
      let dayResponse = InitResponse(
        settings: response.settings,
        categories: response.categories,
        dayRecords: [dayRecord])
      let data = try JSONEncoder.widgetEncoder.encode(
        CachedInitResponse(calendarDate: dayRecord.calendarDate, response: dayResponse))
      try data.write(to: cacheURL(calendarDate: dayRecord.calendarDate), options: .atomic)
      WidgetLogger.debug("Bootstrap cache saved", context: ["calendarDate": dayRecord.calendarDate])
    }
  }

  func removeAll() throws {
    lock.lock()
    defer { lock.unlock() }

    guard FileManager.default.fileExists(atPath: directory.path) else { return }
    for fileURL in try FileManager.default.contentsOfDirectory(
      at: directory, includingPropertiesForKeys: nil)
    {
      try FileManager.default.removeItem(at: fileURL)
    }
    WidgetLogger.debug("Bootstrap cache cleared")
  }

  private func cacheURL(calendarDate: String) -> URL {
    directory.appendingPathComponent("\(calendarDate).json")
  }
}

extension CachedInitResponse {
  fileprivate var dayRecordsContainOnlyCalendarDate: Bool {
    response.dayRecords.count == 1 && response.dayRecords[0].calendarDate == calendarDate
  }
}
