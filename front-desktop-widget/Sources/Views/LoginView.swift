import SwiftUI

struct LoginView: View {
  var authController: AuthController
  @State private var token: String = ""
  @State private var errorMessage: String?

  private let webAppAuthURL = BuildConfig.webAppBaseURL + "/#/token"

  var body: some View {
    VStack(spacing: 12) {
      Text("Todo Planner")
        .font(.system(size: 24, weight: .bold, design: .default))
        .foregroundColor(StyleTokens.primaryText)
        .padding(.top, 8)

      Button(action: {
        openWebAuth()
      }) {
        Text("Open Browser to Login")
          .font(.system(size: 16, weight: .semibold))
          .foregroundColor(.white)
          .frame(maxWidth: .infinity)
          .frame(height: 40)
          .background(Color.blue)
          .cornerRadius(StyleTokens.radiusButton)
      }
      .buttonStyle(.plain)

      TextField("JWT Token", text: $token)
        .textFieldStyle(.plain)
        .padding(8)
        .background(StyleTokens.secondaryText.opacity(0.3))
        .cornerRadius(StyleTokens.radiusButton)
        .foregroundColor(StyleTokens.primaryText)
        .font(.system(size: 14, design: .monospaced))
        .lineLimit(1)
        .truncationMode(.middle)

      Button(action: {
        setToken()
      }) {
        Text("Confirm")
          .font(.system(size: 14, weight: .semibold))
          .foregroundColor(.white)
          .frame(maxWidth: .infinity)
          .frame(height: 40)
          .background(token.isEmpty ? Color.gray : Color.green)
          .cornerRadius(StyleTokens.radiusButton)
      }
      .buttonStyle(.plain)
      .disabled(token.isEmpty)

      if let error = errorMessage {
        Text(error)
          .font(.system(size: 12))
          .foregroundColor(.red)
          .lineLimit(2)
      }

      Spacer()
    }
    .padding(.horizontal, 16)
    .padding(.vertical, 8)
    .frame(width: 320, height: 200)
    .background(StyleTokens.baseVoid)
    .onAppear {
      setupDeepLinkHandler()
    }
  }

  private func setupDeepLinkHandler() {
    WidgetLogger.debug("Setting up deep link handler")

    // Check if there's already a pending token (app was opened via URL before view appeared)
    if let pendingToken = DeepLinkHandler.shared.consumePendingToken() {
      WidgetLogger.debug("Found pending authentication token from deep link")
      self.token = pendingToken
      self.setToken()
      return
    }

    // Set up callback for future tokens
    DeepLinkHandler.shared.onTokenReceived = { receivedToken in
      WidgetLogger.debug("Received authentication token via deep link callback")
      self.token = receivedToken
      self.setToken()
    }
  }

  private func openWebAuth() {
    WidgetLogger.debug("Opening web browser for authentication")
    if let url = URL(string: webAppAuthURL) {
      NSWorkspace.shared.open(url)
      WidgetLogger.debug("Authentication browser opened", context: ["url": webAppAuthURL])
    } else {
      WidgetLogger.error("Invalid web authentication URL", context: ["url": webAppAuthURL])
    }
  }

  private func setToken() {
    guard !token.isEmpty else { return }

    WidgetLogger.debug(
      "Setting authentication token", context: ["length": String(token.count)])

    Task {
      do {
        try await authController.setAuthToken(token)
      } catch {
        WidgetLogger.error(
          "Failed to set authentication token", context: ["error": String(describing: error)])
        errorMessage = "Authentication failed"
      }
    }
  }
}
