import AppKit
import Foundation

class DeepLinkHandler {
  static let shared = DeepLinkHandler()

  private init() {}

  var onTokenReceived: ((String) -> Void)?
  private var pendingToken: String?

  func handleURL(_ url: URL) {
    WidgetLogger.debug("Handling deep link", context: ["url": url.absoluteString])

    guard url.scheme == "todoplanner" else {
      WidgetLogger.error(
        "Deep link has wrong scheme",
        context: ["scheme": url.scheme ?? "nil", "expected": "todoplanner"])
      return
    }

    guard url.host == "auth" else {
      WidgetLogger.error(
        "Deep link has wrong host",
        context: ["host": url.host ?? "nil", "expected": "auth"])
      return
    }

    let components = URLComponents(url: url, resolvingAgainstBaseURL: false)
    if let token = components?.queryItems?.first(where: { $0.name == "token" })?.value {
      WidgetLogger.debug("Authentication token extracted from deep link")

      if let callback = onTokenReceived {
        WidgetLogger.debug("Calling deep link token callback immediately")
        callback(token)
      } else {
        WidgetLogger.debug("No deep link token callback; storing pending token")
        pendingToken = token
      }
    } else {
      WidgetLogger.error("No authentication token found in deep link")
    }
  }

  func consumePendingToken() -> String? {
    WidgetLogger.debug(
      "Checking for pending authentication token",
      context: ["status": pendingToken != nil ? "found" : "none"])
    let token = pendingToken
    pendingToken = nil
    return token
  }
}
