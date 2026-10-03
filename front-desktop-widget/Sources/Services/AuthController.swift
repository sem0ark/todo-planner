import AppKit
import Foundation

/// Controller for authentication state and operations
@MainActor
final class AuthController {
  private let repository: TodoPlannerRepository

  private var webAuthContinuation: CheckedContinuation<Bool, Never>?

  init(repository: TodoPlannerRepository) {
    self.repository = repository
  }

  var hasWorkingAuthenticationToken: Bool {
    guard let token = repository.getAuthToken() else { return false }
    return Self.isWorkingAuthenticationToken(token)
  }

  static func isWorkingAuthenticationToken(_ token: String) -> Bool {
    let tokenParts = token.split(separator: ".")
    guard tokenParts.count == 3 else {
      // Keep supporting non-JWT tokens used by local/test repositories.
      return true
    }

    var payload = String(tokenParts[1])
      .replacingOccurrences(of: "-", with: "+")
      .replacingOccurrences(of: "_", with: "/")
    payload += String(repeating: "=", count: (4 - payload.count % 4) % 4)

    guard let payloadData = Data(base64Encoded: payload),
      let payloadObject = try? JSONSerialization.jsonObject(with: payloadData) as? [String: Any],
      let expiration = payloadObject["exp"] as? TimeInterval
    else { return false }

    return expiration > Date().timeIntervalSince1970
  }

  /// Stores the authentication token for the current process only.
  func setAuthToken(_ token: String) async throws {
    try await repository.persistAuthToken(token)
    WidgetLogger.debug("Authentication successful")
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
            self.webAuthContinuation?.resume(returning: false)
          }
          self.webAuthContinuation = nil
        }
      }

      let authenticationURL = BuildConfig.webAppBaseURL + "/#/token"
      guard let url = URL(string: authenticationURL) else {
        webAuthContinuation?.resume(returning: false)
        webAuthContinuation = nil
        return
      }
      NSWorkspace.shared.open(url)
    }
  }

}
