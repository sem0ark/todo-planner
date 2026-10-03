import Foundation

enum ContentViewMode: Equatable {
  case checkingAuthentication
  case widget
  case login
}

func contentViewMode(
  isCheckingAuthentication: Bool,
  isAuthenticated: Bool,
  hasCachedBootstrap: Bool
) -> ContentViewMode {
  if isCheckingAuthentication {
    return .checkingAuthentication
  }
  if isAuthenticated {
    return .widget
  }
  return .login
}
