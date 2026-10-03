import Foundation

enum APIError: Error {
  case invalidURL
  case networkError(Error)
  case invalidResponse
  case unauthorized
  case serverError(Int, String)
  case decodingError(Error)
}

protocol TodoPlannerAPI: Sendable {
  var authToken: String? { get }
  func setAuthToken(_ token: String)
  func clearAuthToken()
  func validateToken() async throws -> Bool
  func initialize(calendarDate: String) async throws -> InitResponse
  func postDayEvents(date: String, events: [DayEvent]) async throws -> DayEventsResponse
}

final class APIClient: @unchecked Sendable, TodoPlannerAPI {
  static let shared = APIClient()

  private let baseURL: String
  private(set) var authToken: String?
  private let deviceKey = "com.todoplanner.widget.device_id"
  private var deviceId: Int?

  private init() {
    // Load API_BASE_URL from build configuration (set via Makefile)
    // Usage: make build API_BASE_URL=https://api.example.com
    self.baseURL = BuildConfig.apiBaseURL

    WidgetLogger.debug("API base URL configured", context: ["baseURL": self.baseURL])

    let storedDevice = UserDefaults.standard.object(forKey: deviceKey)
    if let storedDeviceId = storedDevice as? Int, storedDeviceId > 0 {
      self.deviceId = storedDeviceId
    } else if storedDevice != nil {
      WidgetLogger.error(
        "Ignoring malformed stored device ID", context: ["value": String(describing: storedDevice)])
      UserDefaults.standard.removeObject(forKey: deviceKey)
    }
  }

  func setAuthToken(_ token: String) {
    self.authToken = token
    self.deviceId = nil
    UserDefaults.standard.removeObject(forKey: deviceKey)
    WidgetLogger.debug("Authentication token stored in memory only")
  }

  func clearAuthToken() {
    self.authToken = nil
    self.deviceId = nil
    UserDefaults.standard.removeObject(forKey: deviceKey)
    WidgetLogger.debug("Cleared in-memory authentication token")
  }

  func hasAuthToken() -> Bool {
    return authToken != nil
  }

  // MARK: - Token Validation

  /// Validates if the current token is still valid by checking device registration
  func validateToken() async throws -> Bool {
    guard authToken != nil else {
      WidgetLogger.debug("No authentication token to validate")
      return false
    }

    do {
      // Validate by attempting device registration or init
      _ = try await registerDeviceIfNeeded()
      WidgetLogger.debug("Authentication token is valid")
      return true
    } catch APIError.unauthorized {
      WidgetLogger.debug("Authentication token is invalid or expired")
      clearAuthToken()
      return false
    } catch {
      WidgetLogger.error("Token validation failed", context: ["error": String(describing: error)])
      throw error
    }
  }

  private func makeRequest<T: Decodable>(
    endpoint: String,
    method: String = "GET",
    body: Encodable? = nil
  ) async throws -> T {
    WidgetLogger.debug(
      "API request started", context: ["method": method, "endpoint": endpoint])

    guard let url = URL(string: baseURL + endpoint) else {
      WidgetLogger.error("Invalid API URL", context: ["endpoint": endpoint])
      throw APIError.invalidURL
    }

    var request = URLRequest(url: url)
    request.timeoutInterval = 15
    request.httpMethod = method
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")

    if let token = authToken {
      WidgetLogger.debug("Authorization header configured")
      request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    } else {
      WidgetLogger.debug("No authentication token set")
    }

    if let body = body {
      let encoder = JSONEncoder()
      encoder.dateEncodingStrategy = .iso8601
      do {
        request.httpBody = try encoder.encode(body)
        WidgetLogger.debug(
          "Request body encoded", context: ["bytes": String(request.httpBody?.count ?? 0)])
      } catch {
        WidgetLogger.error(
          "Failed to encode request body", context: ["error": String(describing: error)])
        throw error
      }
    }

    do {
      WidgetLogger.debug("Sending API request")
      let (data, response) = try await URLSession.shared.data(for: request)

      guard let httpResponse = response as? HTTPURLResponse else {
        WidgetLogger.error("Invalid API response type")
        throw APIError.invalidResponse
      }

      WidgetLogger.debug(
        "API response received", context: ["statusCode": String(httpResponse.statusCode)])

      WidgetLogger.debug("API response body received", context: ["bytes": String(data.count)])

      switch httpResponse.statusCode {
      case 200...299:
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        do {
          let decoded = try decoder.decode(T.self, from: data)
          WidgetLogger.debug("API response decoded successfully")
          return decoded
        } catch {
          WidgetLogger.error(
            "API response decoding failed", context: ["error": String(describing: error)])
          if let decodingError = error as? DecodingError {
            switch decodingError {
            case .keyNotFound(let key, let context):
              WidgetLogger.error(
                "Missing response key",
                context: ["key": key.stringValue, "details": context.debugDescription])
            case .typeMismatch(let type, let context):
              WidgetLogger.error(
                "Response type mismatch",
                context: ["type": String(describing: type), "details": context.debugDescription])
            case .valueNotFound(let type, let context):
              WidgetLogger.error(
                "Response value missing",
                context: ["type": String(describing: type), "details": context.debugDescription])
            case .dataCorrupted(let context):
              WidgetLogger.error(
                "Response data corrupted", context: ["details": context.debugDescription])
            @unknown default:
              WidgetLogger.error("Unknown response decoding error")
            }
          }
          throw APIError.decodingError(error)
        }
      case 401:
        WidgetLogger.error("API request unauthorized", context: ["statusCode": "401"])
        clearAuthToken()
        throw APIError.unauthorized
      default:
        let errorMessage = String(data: data, encoding: .utf8) ?? "Unknown error"
        WidgetLogger.error(
          "API server error",
          context: ["statusCode": String(httpResponse.statusCode), "message": errorMessage])
        throw APIError.serverError(httpResponse.statusCode, errorMessage)
      }
    } catch let error as APIError {
      throw error
    } catch {
      WidgetLogger.error("API network error", context: ["error": String(describing: error)])
      throw APIError.networkError(error)
    }
  }

  func initialize(calendarDate: String) async throws -> InitResponse {
    let registeredDeviceId = try await registerDeviceIfNeeded()
    struct InitRequest: Encodable {
      let device_id: Int
      let calendar_date: String
    }

    do {
      return try await makeRequest(
        endpoint: "/init",
        method: "POST",
        body: InitRequest(device_id: registeredDeviceId, calendar_date: calendarDate)
      )
    } catch APIError.serverError(404, let message) where message.contains("device not found") {
      WidgetLogger.error(
        "Registered device was rejected by server",
        context: ["calendarDate": calendarDate, "deviceId": String(registeredDeviceId)])
      clearDeviceRegistration()
      let replacementDeviceId = try await registerDeviceIfNeeded()
      return try await makeRequest(
        endpoint: "/init",
        method: "POST",
        body: InitRequest(device_id: replacementDeviceId, calendar_date: calendarDate)
      )
    }
  }

  private func registerDeviceIfNeeded() async throws -> Int {
    if let deviceId { return deviceId }

    struct DeviceRequest: Encodable { let platform: String }
    let registration: DeviceRegistration = try await makeRequest(
      endpoint: "/devices",
      method: "POST",
      body: DeviceRequest(platform: "desktop")
    )
    deviceId = registration.deviceId
    UserDefaults.standard.set(registration.deviceId, forKey: deviceKey)
    return registration.deviceId
  }

  private func clearDeviceRegistration() {
    deviceId = nil
    UserDefaults.standard.removeObject(forKey: deviceKey)
  }

  func postDayEvents(
    date: String,
    events: [DayEvent]
  ) async throws -> DayEventsResponse {
    let registeredDeviceId = try await registerDeviceIfNeeded()
    do {
      let request = DayEventsRequest(deviceId: registeredDeviceId, events: events)
      return try await makeRequest(
        endpoint: "/days/\(date)/events",
        method: "POST",
        body: request
      )
    } catch APIError.serverError(404, let message) where message.contains("device not found") {
      clearDeviceRegistration()
      let replacementDeviceId = try await registerDeviceIfNeeded()
      let request = DayEventsRequest(deviceId: replacementDeviceId, events: events)
      return try await makeRequest(
        endpoint: "/days/\(date)/events",
        method: "POST",
        body: request
      )
    }
  }

}
