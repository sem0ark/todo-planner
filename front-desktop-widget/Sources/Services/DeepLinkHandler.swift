import Foundation
import AppKit

class DeepLinkHandler {
    static let shared = DeepLinkHandler()

    private init() {}

    var onTokenReceived: ((String) -> Void)?
    private var pendingToken: String?

    func handleURL(_ url: URL) {
        print("[DEEPLINK] Handling URL: \(url.absoluteString)")

        guard url.scheme == "todoplanner" else {
            print("[DEEPLINK] Wrong scheme: \(url.scheme ?? "nil"), expected 'todoplanner'")
            return
        }

        guard url.host == "auth" else {
            print("[DEEPLINK] Wrong host: \(url.host ?? "nil"), expected 'auth'")
            return
        }

        let components = URLComponents(url: url, resolvingAgainstBaseURL: false)
        if let token = components?.queryItems?.first(where: { $0.name == "token" })?.value {
            print("[DEEPLINK] Extracted token: \(String(token.prefix(20)))...")

            if let callback = onTokenReceived {
                print("[DEEPLINK] Calling onTokenReceived callback immediately")
                callback(token)
            } else {
                print("[DEEPLINK] No callback set yet, storing pending token")
                pendingToken = token
            }
        } else {
            print("[DEEPLINK] No token found in URL query parameters")
        }
    }

    func consumePendingToken() -> String? {
        print("[DEEPLINK] Checking for pending token: \(pendingToken != nil ? "found" : "none")")
        let token = pendingToken
        pendingToken = nil
        return token
    }
}
