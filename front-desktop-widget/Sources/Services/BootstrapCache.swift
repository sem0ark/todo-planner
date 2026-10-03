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
        cachedResponse.response.dayRecord.calendarDate == calendarDate
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
    guard response.dayRecord.calendarDate == calendarDate else {
      throw StorageError.invalidContract("bootstrap calendar date does not match request")
    }

    lock.lock()
    defer { lock.unlock() }

    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    let data = try JSONEncoder.widgetEncoder.encode(
      CachedInitResponse(calendarDate: calendarDate, response: response))
    try data.write(to: cacheURL(calendarDate: calendarDate), options: .atomic)
    WidgetLogger.debug("Bootstrap cache saved", context: ["calendarDate": calendarDate])
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
