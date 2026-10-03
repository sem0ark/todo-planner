import AppKit
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
  private var webAuthContinuation: CheckedContinuation<Bool, Never>?

  init(repository: TodoPlannerRepository) {
    self.repository = repository
  }

  /// Restores the in-memory session without making a network request.
  func checkInitialAuth() async {
    isAuthenticated = repository.getAuthToken() != nil
    lastError = nil
    isCheckingAuth = false
    WidgetLogger.debug(
      "Initial authentication state loaded from memory",
      context: ["hasToken": String(isAuthenticated)])
  }

  /// Stores the authentication token for the current process only.
  func setAuthToken(_ token: String) async throws {
    try await repository.persistAuthToken(token)
    isAuthenticated = true
    lastError = nil
    print("[OK] Authentication successful")
  }

  func authenticateFromWeb() async -> Bool {
    WidgetLogger.debug("Opening web authentication flow")
    return await withCheckedContinuation { continuation in
      webAuthContinuation?.resume(returning: false)
      webAuthContinuation = continuation

      DeepLinkHandler.shared.onTokenReceived = { [weak self] receivedToken in
        Task { @MainActor [weak self] in
          guard let self else { return }
          do {
            try await self.setAuthToken(receivedToken)
            WidgetLogger.debug("Web authentication token received")
            self.webAuthContinuation?.resume(returning: true)
          } catch {
            self.lastError = String(describing: error)
            self.webAuthContinuation?.resume(returning: false)
          }
          self.webAuthContinuation = nil
        }
      }

      let authenticationURL = BuildConfig.webAppBaseURL + "/#/token"
      guard let url = URL(string: authenticationURL) else {
        lastError = "Invalid web authentication URL"
        webAuthContinuation?.resume(returning: false)
        webAuthContinuation = nil
        return
      }
      NSWorkspace.shared.open(url)
    }
  }

  /// Clear authentication and logout
  func handleLogout() async {
    WidgetLogger.debug("Logging out...")
    print("[AUTH] Logging out...")
    do {
      try await repository.clearAuth()
      isAuthenticated = false
      lastError = nil
      print("[OK] Logout successful")
    } catch {
      WidgetLogger.error("Logout failed", context: ["error": String(describing: error)])
      lastError = String(describing: error)
    }
  }
}
