import Foundation
import Observation
import SwiftUI

/// Controller for authentication state and operations
@Observable
@MainActor
final class AuthController {
  private let repository: TodoPlannerRepository

  var isAuthenticated: Bool = false
  var isCheckingAuth: Bool = true
  var lastError: String?

  init(repository: TodoPlannerRepository) {
    self.repository = repository
  }

  /// Check if there's a saved token and validate it
  func checkInitialAuth() async {
    print("[AUTH] Checking for saved token...")

    guard repository.getAuthToken() != nil else {
      print("[AUTH] No token found")
      isAuthenticated = false
      lastError = nil
      isCheckingAuth = false
      return
    }

    print("[AUTH] Token found, validating...")
    do {
      let isValid = try await repository.validateAuth()
      if isValid {
        print("[OK] Token is valid, user authenticated")
        isAuthenticated = true
        lastError = nil
      } else {
        print("[AUTH] Token invalid")
        isAuthenticated = false
        lastError = nil
      }
    } catch {
      WidgetLogger.error(
        "Authentication validation failed", context: ["error": String(describing: error)])
      lastError = String(describing: error)
    }

    isCheckingAuth = false
  }

  /// Persist authentication token
  func setAuthToken(_ token: String) async throws {
    try await repository.persistAuthToken(token)
    isAuthenticated = true
    lastError = nil
    print("[OK] Authentication successful!")
  }

  /// Clear authentication and logout
  func handleLogout() async {
    print("[AUTH] Logging out...")
    do {
      try await repository.clearAuth()
      isAuthenticated = false
      print("[OK] Logout successful")
    } catch {
      WidgetLogger.error("Logout failed", context: ["error": String(describing: error)])
      lastError = String(describing: error)
    }
  }
}
