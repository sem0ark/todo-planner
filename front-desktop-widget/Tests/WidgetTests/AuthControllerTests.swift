import Foundation

@MainActor
final class AuthControllerTests: WidgetTestCase {
  func test_workingAuthenticationToken_hasFutureExpiration() throws {
    let token = try makeToken(expiration: Date().addingTimeInterval(60))

    try assert(
      AuthController.isWorkingAuthenticationToken(token),
      "A token with a future expiration should be working")
  }

  func test_expiredAuthenticationToken_isNotWorking() throws {
    let token = try makeToken(expiration: Date().addingTimeInterval(-60))

    try assert(
      !AuthController.isWorkingAuthenticationToken(token),
      "A token with a past expiration should not be working")
  }

  func test_malformedJwt_isNotWorking() throws {
    try assert(
      !AuthController.isWorkingAuthenticationToken("header.payload.signature"),
      "A JWT with an invalid payload should not be working")
  }

  func test_nonJwtToken_isSupported() throws {
    try assert(
      AuthController.isWorkingAuthenticationToken("test-token"),
      "Non-JWT tokens used by local repositories should remain supported")
  }

  private func makeToken(expiration: Date) throws -> String {
    let headerData = Data(#"{"alg":"none","typ":"JWT"}"#.utf8)
    let payloadData = try JSONSerialization.data(
      withJSONObject: ["exp": expiration.timeIntervalSince1970])
    let header = base64URLData(headerData)
    let payload = base64URLData(payloadData)
    return "\(header).\(payload).signature"
  }

  private func base64URLData(_ data: Data) -> String {
    data.base64EncodedString()
      .replacingOccurrences(of: "+", with: "-")
      .replacingOccurrences(of: "/", with: "_")
      .replacingOccurrences(of: "=", with: "")
  }

  static func testMethods() -> [TestCase] {
    let tests = AuthControllerTests()
    return [
      ("test_workingAuthenticationToken_hasFutureExpiration", {
        try tests.test_workingAuthenticationToken_hasFutureExpiration()
      }),
      ("test_expiredAuthenticationToken_isNotWorking", {
        try tests.test_expiredAuthenticationToken_isNotWorking()
      }),
      ("test_malformedJwt_isNotWorking", {
        try tests.test_malformedJwt_isNotWorking()
      }),
      ("test_nonJwtToken_isSupported", {
        try tests.test_nonJwtToken_isSupported()
      }),
    ]
  }
}
