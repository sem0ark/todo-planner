import Foundation
import SwiftUI

/// Controller for authentication state and operations
@MainActor
final class AuthController: ObservableObject {
  private let repository: TodoPlannerRepository

  @Published var isAuthenticated: Bool = false
  @Published var isCheckingAuth: Bool = true

  init(repository: TodoPlannerRepository) {
    self.repository = repository
  }

  /// Check if there's a saved token and validate it
  func checkInitialAuth() async {
    print("[AUTH] Checking for saved token...")

    guard repository.getAuthToken() != nil else {
      print("[AUTH] No token found")
      isAuthenticated = false
      isCheckingAuth = false
      return
    }

    print("[AUTH] Token found, validating...")
    do {
      let isValid = try await repository.validateAuth()
      if isValid {
        print("[OK] Token is valid, user authenticated")
        isAuthenticated = true
      } else {
        print("[AUTH] Token invalid")
        isAuthenticated = false
      }
    } catch {
      print("[AUTH] Validation failed: \(error)")
      isAuthenticated = false
    }

    isCheckingAuth = false
  }

  /// Persist authentication token
  func setAuthToken(_ token: String) async throws {
    try await repository.persistAuthToken(token)
    isAuthenticated = true
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
      print("[ERROR] Logout failed: \(error)")
    }
  }
}
