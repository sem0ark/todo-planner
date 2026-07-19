import SwiftUI

struct LoginView: View {
    @Binding var isAuthenticated: Bool
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
        print("[LOGIN] Setting up deep link handler")

        // Check if there's already a pending token (app was opened via URL before view appeared)
        if let pendingToken = DeepLinkHandler.shared.consumePendingToken() {
            print("[LOGIN] Found pending token from deep link, using it now")
            self.token = pendingToken
            self.setToken()
            return
        }

        // Set up callback for future tokens
        DeepLinkHandler.shared.onTokenReceived = { receivedToken in
            print("[LOGIN] Received token via callback")
            self.token = receivedToken
            self.setToken()
        }
    }

    private func openWebAuth() {
        print("[AUTH] Opening web browser for authentication...")
        if let url = URL(string: webAppAuthURL) {
            NSWorkspace.shared.open(url)
            print("[AUTH] Browser opened: \(webAppAuthURL)")
        } else {
            print("[ERROR] Invalid web app URL: \(webAppAuthURL)")
        }
    }

    private func setToken() {
        guard !token.isEmpty else { return }

        print("[AUTH] Setting authentication token...")
        print("[AUTH] Token length: \(token.count) characters")
        APIClient.shared.setAuthToken(token)
        isAuthenticated = true
        print("[OK] Authentication successful!")
    }
}
