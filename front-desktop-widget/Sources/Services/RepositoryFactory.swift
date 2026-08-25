import Foundation

/// Factory for creating the appropriate repository based on build configuration
enum RepositoryFactory {
  /// Creates a repository instance based on the STORAGE_MODE build configuration
  static func createRepository() -> TodoPlannerRepository {
    let mode = BuildConfig.storageMode

    print("[FACTORY] Storage mode: \(mode)")

    switch mode {
    case "mock":
      print("[FACTORY] Using MockTodoPlannerRepository (in-memory)")
      return MockTodoPlannerRepository()
    case "local":
      print("[FACTORY] Using LocalTodoPlannerRepository (SQLite)")
      let dbPath = getLocalDatabasePath()
      do {
        return try LocalTodoPlannerRepository(dbPath: dbPath)
      } catch {
        print("[FACTORY] Failed to initialize LocalTodoPlannerRepository: \(error)")
        print("[FACTORY] Falling back to MockTodoPlannerRepository")
        return MockTodoPlannerRepository()
      }
    case "remote":
      print("[FACTORY] Using RemoteTodoPlannerRepository (API)")
      return RemoteTodoPlannerRepository()
    default:
      print("[FACTORY] Unknown mode '\(mode)', defaulting to remote")
      return RemoteTodoPlannerRepository()
    }
  }

  private static func getLocalDatabasePath() -> String {
    let appSupport = FileManager.default.urls(
      for: .applicationSupportDirectory, in: .userDomainMask
    ).first!
    let appDir = appSupport.appendingPathComponent("TodoPlannerWidget", isDirectory: true)

    // Create directory if it doesn't exist
    try? FileManager.default.createDirectory(at: appDir, withIntermediateDirectories: true)

    let dbPath = appDir.appendingPathComponent("todoplanner.db").path
    print("[FACTORY] Local DB path: \(dbPath)")
    return dbPath
  }
}
