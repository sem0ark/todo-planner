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

  func test_cachedBootstrapShowsWidgetWithoutAuthentication() throws {
    try assertEqual(
      contentViewMode(
        isCheckingAuthentication: false,
        isAuthenticated: false,
        hasCachedBootstrap: true),
      .widget)
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
}
