import Foundation

@MainActor
final class DeepLinkHandlerTests: WidgetTestCase {
  func test_tokenReceivedBeforeCallback_isBufferedForLaterConsumption() throws {
    DeepLinkHandler.shared.onTokenReceived = nil
    _ = DeepLinkHandler.shared.consumePendingToken()

    let url = try requireURL("todoplanner://auth?token=early-token")
    DeepLinkHandler.shared.handleURL(url)

    try assertEqual(DeepLinkHandler.shared.consumePendingToken(), "early-token")
    try assertEqual(DeepLinkHandler.shared.consumePendingToken(), nil)
  }

  private func requireURL(_ string: String) throws -> URL {
    guard let url = URL(string: string) else {
      throw AssertionError.failed("Expected a valid deep-link URL")
    }
    return url
  }

  static func testMethods() -> [TestCase] {
    let tests = DeepLinkHandlerTests()
    return [
      (
        "test_tokenReceivedBeforeCallback_isBufferedForLaterConsumption",
        {
          try tests.test_tokenReceivedBeforeCallback_isBufferedForLaterConsumption()
        }
      )
    ]
  }
}
