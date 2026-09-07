import Foundation

enum APIError: Error {
  case invalidURL
  case networkError(Error)
  case invalidResponse
  case unauthorized
  case serverError(Int, String)
  case decodingError(Error)
}

final class APIClient: @unchecked Sendable {
  static let shared = APIClient()

  private let baseURL: String
  private var authToken: String?
  private let tokenKey = "com.todoplanner.widget.jwt_token"
  private let deviceKey = "com.todoplanner.widget.device_id"
  private var initializationCache: [String: InitResponse] = [:]
  private var deviceId: Int?

  var currentDeviceId: Int { deviceId ?? 0 }

  private init() {
    // Load API_BASE_URL from build configuration (set via Makefile)
    // Usage: make build API_BASE_URL=https://api.example.com
    self.baseURL = BuildConfig.apiBaseURL

    print("[CONFIG] API Base URL: \(self.baseURL)")

    // Try to load token from UserDefaults on init
    if let savedToken = loadTokenFromAppData() {
      print("[AUTH] Loaded JWT from app data")
      self.authToken = savedToken
    } else {
      print("[AUTH] No saved JWT found in app data")
    }

    self.deviceId = UserDefaults.standard.object(forKey: deviceKey) as? Int
  }

  func setAuthToken(_ token: String) {
    self.authToken = token
    self.deviceId = nil
    self.initializationCache.removeAll()
    UserDefaults.standard.removeObject(forKey: deviceKey)
    saveTokenToAppData(token)
  }

  func clearAuthToken() {
    self.authToken = nil
    self.deviceId = nil
    self.initializationCache.removeAll()
    UserDefaults.standard.removeObject(forKey: deviceKey)
    deleteTokenFromAppData()
  }

  func hasAuthToken() -> Bool {
    return authToken != nil
  }

  // MARK: - App Data Storage (UserDefaults)
  // TODO: Consider encrypting the token before storing in UserDefaults
  // or using a more secure storage mechanism for production

  private func saveTokenToAppData(_ token: String) {
    UserDefaults.standard.set(token, forKey: tokenKey)
    UserDefaults.standard.synchronize()
    print("[OK] JWT saved to app data")
  }

  private func loadTokenFromAppData() -> String? {
    return UserDefaults.standard.string(forKey: tokenKey)
  }

  private func deleteTokenFromAppData() {
    UserDefaults.standard.removeObject(forKey: tokenKey)
    UserDefaults.standard.synchronize()
    print("[OK] JWT deleted from app data")
  }

  // MARK: - Token Validation

  /// Validates if the current token is still valid by checking device registration
  func validateToken() async -> Bool {
    guard authToken != nil else {
      print("[AUTH] No token to validate")
      return false
    }

    do {
      // Validate by attempting device registration or init
      _ = try await registerDeviceIfNeeded()
      print("[OK] Token is valid")
      return true
    } catch APIError.unauthorized {
      print("[AUTH] Token is invalid or expired")
      clearAuthToken()
      return false
    } catch {
      print("[ERROR] Token validation failed: \(error)")
      return false
    }
  }

  private func makeRequest<T: Decodable>(
    endpoint: String,
    method: String = "GET",
    body: Encodable? = nil
  ) async throws -> T {
    print("[API] API Request: \(method) \(baseURL)\(endpoint)")

    guard let url = URL(string: baseURL + endpoint) else {
      print("[ERROR] Invalid URL: \(baseURL)\(endpoint)")
      throw APIError.invalidURL
    }

    var request = URLRequest(url: url)
    request.httpMethod = method
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")

    if let token = authToken {
      let tokenPreview = String(token.prefix(20))
      print("[AUTH] Authorization: Bearer \(tokenPreview)...")
      request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    } else {
      print("[WARN] No auth token set")
    }

    if let body = body {
      let encoder = JSONEncoder()
      encoder.dateEncodingStrategy = .iso8601
      do {
        request.httpBody = try encoder.encode(body)
        if let bodyString = String(data: request.httpBody!, encoding: .utf8) {
          print("[OUT] Request body: \(bodyString)")
        }
      } catch {
        print("[ERROR] Failed to encode request body: \(error)")
        throw error
      }
    }

    do {
      print("[WAIT] Sending request...")
      let (data, response) = try await URLSession.shared.data(for: request)

      guard let httpResponse = response as? HTTPURLResponse else {
        print("[ERROR] Invalid response type")
        throw APIError.invalidResponse
      }

      print("[IN] Response status: \(httpResponse.statusCode)")

      let responseString = String(data: data, encoding: .utf8) ?? ""
      print("[IN] Response body (\(data.count) bytes): '\(responseString)'")

      switch httpResponse.statusCode {
      case 200...299:
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        do {
          let decoded = try decoder.decode(T.self, from: data)
          print("[OK] Successfully decoded response")
          return decoded
        } catch {
          print("[ERROR] Decoding error: \(error)")
          if let decodingError = error as? DecodingError {
            switch decodingError {
            case .keyNotFound(let key, let context):
              print("   Missing key '\(key.stringValue)' - \(context.debugDescription)")
            case .typeMismatch(let type, let context):
              print("   Type mismatch for type '\(type)' - \(context.debugDescription)")
            case .valueNotFound(let type, let context):
              print("   Value not found for type '\(type)' - \(context.debugDescription)")
            case .dataCorrupted(let context):
              print("   Data corrupted - \(context.debugDescription)")
            @unknown default:
              print("   Unknown decoding error")
            }
          }
          throw APIError.decodingError(error)
        }
      case 401:
        print("[ERROR] Unauthorized (401)")
        throw APIError.unauthorized
      default:
        let errorMessage = String(data: data, encoding: .utf8) ?? "Unknown error"
        print("[ERROR] Server error \(httpResponse.statusCode): \(errorMessage)")
        throw APIError.serverError(httpResponse.statusCode, errorMessage)
      }
    } catch let error as APIError {
      throw error
    } catch {
      print("[ERROR] Network error: \(error)")
      throw APIError.networkError(error)
    }
  }

  func initialize(calendarDate: String) async throws -> InitResponse {
    if let cachedResponse = initializationCache[calendarDate] {
      return cachedResponse
    }
    let currentDeviceId = try await registerDeviceIfNeeded()
    struct InitRequest: Encodable {
      let device_id: Int
      let calendar_date: String
    }

    let response: InitResponse
    do {
      response = try await makeRequest(
        endpoint: "/init",
        method: "POST",
        body: InitRequest(device_id: currentDeviceId, calendar_date: calendarDate)
      )
    } catch APIError.serverError(404, let message) where message.contains("device not found") {
      deviceId = nil
      UserDefaults.standard.removeObject(forKey: deviceKey)
      let replacementDeviceId = try await registerDeviceIfNeeded()
      response = try await makeRequest(
        endpoint: "/init",
        method: "POST",
        body: InitRequest(device_id: replacementDeviceId, calendar_date: calendarDate)
      )
    }
    initializationCache[calendarDate] = response
    return response
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

  func postDayEvents(
    date: String,
    events: [DayEvent]
  ) async throws -> DayEventsResponse {
    let request = DayEventsRequest(deviceId: try await registerDeviceIfNeeded(), events: events)
    return try await makeRequest(
      endpoint: "/days/\(date)/events",
      method: "POST",
      body: request
    )
  }

}
