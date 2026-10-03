import Foundation

/// Factory for creating the appropriate repository based on build configuration
enum RepositoryFactory {
  /// Creates a repository instance based on the STORAGE_MODE build configuration
  static func createRepository() -> TodoPlannerRepository {
    let mode = BuildConfig.storageMode

    WidgetLogger.debug("Storage mode selected", context: ["mode": mode])

    switch mode {
    case "mock":
      WidgetLogger.debug("Using mock repository")
      return MockTodoPlannerRepository()
    case "remote":
      WidgetLogger.debug("Using remote repository")
      return RemoteTodoPlannerRepository()
    default:
      WidgetLogger.error(
        "Unknown storage mode; defaulting to remote", context: ["mode": mode])
      return RemoteTodoPlannerRepository()
    }
  }

}
