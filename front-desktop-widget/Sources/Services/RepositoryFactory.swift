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
    case "remote":
      print("[FACTORY] Using RemoteTodoPlannerRepository (API)")
      return RemoteTodoPlannerRepository()
    default:
      print("[FACTORY] Unknown mode '\(mode)', defaulting to remote")
      return RemoteTodoPlannerRepository()
    }
  }

}
