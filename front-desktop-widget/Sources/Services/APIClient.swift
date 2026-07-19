import Foundation

enum APIError: Error {
    case invalidURL
    case networkError(Error)
    case invalidResponse
    case unauthorized
    case serverError(Int, String)
    case decodingError(Error)
}

class APIClient {
    static let shared = APIClient()

    private let baseURL: String
    private var authToken: String?

    private init() {
        // TODO: Load from configuration or environment
        // self.baseURL = ProcessInfo.processInfo.environment["API_BASE_URL"] ?? "http://localhost:8080"
        self.baseURL = "http://localhost:8080"
    }

    func setAuthToken(_ token: String) {
        self.authToken = token
    }

    func clearAuthToken() {
        self.authToken = nil
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

            if let responseString = String(data: data, encoding: .utf8) {
                print("[IN] Response body: \(responseString)")
            }

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

    func login(username: String, password: String) async throws -> (token: String, userId: Int) {
        struct LoginRequest: Encodable {
            let username: String
            let password: String
        }

        struct LoginResponse: Decodable {
            let token: String
            let user_id: Int
        }

        let response: LoginResponse = try await makeRequest(
            endpoint: "/auth/login",
            method: "POST",
            body: LoginRequest(username: username, password: password)
        )

        setAuthToken(response.token)
        return (token: response.token, userId: response.user_id)
    }

    func fetchCategories() async throws -> [Category] {
        let response: CategoriesResponse = try await makeRequest(endpoint: "/categories")
        return response.categories
    }

    func fetchDayRecords(from: String, to: String) async throws -> [DayRecord] {
        let response: DayRecordsResponse = try await makeRequest(
            endpoint: "/day-records?from=\(from)&to=\(to)"
        )
        return response.dayRecords
    }

    func createDayRecord(date: String) async throws -> DayRecord {
        struct CreateDayRecordRequest: Encodable {
            let calendar_date: String
        }

        return try await makeRequest(
            endpoint: "/day-records",
            method: "POST",
            body: CreateDayRecordRequest(calendar_date: date)
        )
    }

    func postDayEvents(dayRecordId: Int, events: [DayEvent]) async throws -> DayEventsResponse {
        let request = DayEventsRequest(events: events)
        return try await makeRequest(
            endpoint: "/day-records/\(dayRecordId)/events",
            method: "POST",
            body: request
        )
    }
}
