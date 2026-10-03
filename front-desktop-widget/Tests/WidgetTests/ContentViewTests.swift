import Foundation

@MainActor
final class ContentViewTests {
  func test_checkingAuthenticationTakesPriority() throws {
    try assertEqual(
      contentViewMode(
        isCheckingAuthentication: true,
        isAuthenticated: false,
        hasCachedBootstrap: true),
      .checkingAuthentication)
  }

  func test_cachedBootstrapStillRequiresAuthentication() throws {
    try assertEqual(
      contentViewMode(
        isCheckingAuthentication: false,
        isAuthenticated: false,
        hasCachedBootstrap: true),
      .login)
  }

  func test_authenticatedSessionShowsWidgetWithoutCache() throws {
    try assertEqual(
      contentViewMode(
        isCheckingAuthentication: false,
        isAuthenticated: true,
        hasCachedBootstrap: false),
      .widget)
  }

  func test_missingSessionAndCacheShowsLogin() throws {
    try assertEqual(
      contentViewMode(
        isCheckingAuthentication: false,
        isAuthenticated: false,
        hasCachedBootstrap: false),
      .login)
  }
  static func testMethods() -> [TestCase] {
    let tests = ContentViewTests()
    return [
      ("test_checkingAuthenticationTakesPriority", { try tests.test_checkingAuthenticationTakesPriority() }),
      ("test_cachedBootstrapStillRequiresAuthentication", { try tests.test_cachedBootstrapStillRequiresAuthentication() }),
      ("test_authenticatedSessionShowsWidgetWithoutCache", { try tests.test_authenticatedSessionShowsWidgetWithoutCache() }),
      ("test_missingSessionAndCacheShowsLogin", { try tests.test_missingSessionAndCacheShowsLogin() }),
    ]
  }
}
