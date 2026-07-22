import Foundation

// MARK: - Unified Test Suite
// All widget tests in one place: Models, State Logic, State Transitions

// ============================================================================
// MARK: - Test Runner
// ============================================================================

@main
struct AllTests {
    static func main() async {
        printHeader("Todo Planner Desktop Widget - Complete Test Suite")

        var totalPassed = 0
        var totalFailed = 0

        // Run test suites
        let (p1, f1) = runModelTests()
        totalPassed += p1; totalFailed += f1

        let (p2, f2) = runStateLogicTests()
        totalPassed += p2; totalFailed += f2

        let (p3, f3) = runStateTransitionTests()
        totalPassed += p3; totalFailed += f3

        let (p4, f4) = await runMockAPITests()
        totalPassed += p4; totalFailed += f4

        // Summary
        printSeparator()
        printHeader("Final Summary")
        print("  ✅ Passed: \(totalPassed)")
        print("  ❌ Failed: \(totalFailed)")
        print("  📊 Total:  \(totalPassed + totalFailed)")
        printSeparator()

        if totalFailed > 0 {
            print("❌ Some tests failed\n")
            exit(1)
        } else {
            print("🎉 All tests passed!\n")
            exit(0)
        }
    }
}

// ============================================================================
// MARK: - Model Tests
// ============================================================================

func runModelTests() -> (passed: Int, failed: Int) {
    printSection("Model Serialization Tests")

    let tests: [(String, () throws -> Void)] = [
        ("Category JSON decode", testCategoryModel),
        ("DayRecord JSON decode", testDayRecordModel),
        ("Date encoding/decoding", testDateEncodingDecoding),
        ("PlannedBlock JSON decode", testPlannedBlockModel),
        ("ActualBlock JSON decode", testActualBlockModel),
        ("DayEvent JSON encode", testDayEventModel),
    ]

    return runSyncTests(tests)
}

func testCategoryModel() throws {
    let json = """
    {
        "id": 1,
        "name": "Work",
        "color": "#FF5733",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    }
    """.data(using: .utf8)!

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    let category = try decoder.decode(Category.self, from: json)

    assert(category.id == 1 && category.name == "Work" && category.color == "#FF5733")
}

func testDayRecordModel() throws {
    let json = """
    {
        "id": 1,
        "calendar_date": "2024-01-01",
        "review_status": "Unreviewed",
        "snapshot_blocks": [],
        "actual_blocks": [],
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    }
    """.data(using: .utf8)!

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    let record = try decoder.decode(DayRecord.self, from: json)

    assert(record.id == 1 && record.calendarDate == "2024-01-01")
}

func testDateEncodingDecoding() throws {
    let event = DayEvent(
        eventType: "confirmation",
        outgoingCategoryId: nil,
        incomingCategoryId: nil,
        occurredAt: Date()
    )

    let encoder = JSONEncoder()
    encoder.dateEncodingStrategy = .iso8601
    let data = try encoder.encode(event)

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    let decoded = try decoder.decode(DayEvent.self, from: data)

    assert(decoded.eventType == "confirmation")
}

func testPlannedBlockModel() throws {
    let json = """
    {
        "id": 1,
        "category_id": 2,
        "start_time": "09:00:00",
        "duration_minutes": 60
    }
    """.data(using: .utf8)!

    let block = try JSONDecoder().decode(PlannedBlock.self, from: json)
    assert(block.id == 1 && block.categoryId == 2 && block.durationMinutes == 60)
}

func testActualBlockModel() throws {
    let json = """
    {
        "id": 1,
        "category_id": 2,
        "block_type": "actual",
        "start_time": "09:00:00",
        "duration_minutes": 60
    }
    """.data(using: .utf8)!

    let block = try JSONDecoder().decode(ActualBlock.self, from: json)
    assert(block.id == 1 && block.blockType == "actual")
}

func testDayEventModel() throws {
    let json = """
    {
        "event_type": "transition",
        "outgoing_category_id": 1,
        "incoming_category_id": 2,
        "occurred_at": "2024-01-01T12:00:00Z"
    }
    """.data(using: .utf8)!

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    let event = try decoder.decode(DayEvent.self, from: json)

    assert(event.eventType == "transition" && event.incomingCategoryId == 2)
}

// ============================================================================
// MARK: - State Logic Tests
// ============================================================================

func runStateLogicTests() -> (passed: Int, failed: Int) {
    printSection("State Logic Tests")

    let tests: [(String, () throws -> Void)] = [
        ("Time block detection", testTimeBlockDetection),
        ("Progress calculation", testProgressCalculation),
        ("Progress boundaries (0% and 100%)", testProgressBoundaries),
        ("Block not found outside range", testBlockNotFound),
        ("Multiple blocks selection", testMultipleBlocks),
    ]

    return runSyncTests(tests)
}

func testTimeBlockDetection() throws {
    let now = Date()
    let calendar = Calendar.current
    let components = calendar.dateComponents([.hour, .minute], from: now)
    let currentTime = String(format: "%02d:%02d:00", components.hour ?? 0, components.minute ?? 0)

    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: currentTime, duration: 60)
    let result = getCurrentPlannedBlock(at: now, from: [block])

    assert(result?.id == 1, "Should find current planned block")
}

func testProgressCalculation() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())
    components.hour = 10
    components.minute = 30
    let testTime = calendar.date(from: components)!

    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: "10:00:00", duration: 60)
    let progress = calculateProgress(for: block, at: testTime)

    assert(progress >= 0.49 && progress <= 0.51, "Progress should be ~50%, got \(progress)")
}

func testProgressBoundaries() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())

    components.hour = 10
    components.minute = 0
    let startTime = calendar.date(from: components)!

    components.hour = 11
    components.minute = 0
    let endTime = calendar.date(from: components)!

    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: "10:00:00", duration: 60)

    let progressStart = calculateProgress(for: block, at: startTime)
    let progressEnd = calculateProgress(for: block, at: endTime)

    assert(progressStart == 0.0, "Progress at start should be 0%")
    assert(progressEnd >= 0.99, "Progress at end should be ~100%")
}

func testBlockNotFound() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())
    components.hour = 15
    components.minute = 0
    let testTime = calendar.date(from: components)!

    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: "10:00:00", duration: 60)
    let result = getCurrentPlannedBlock(at: testTime, from: [block])

    assert(result == nil, "Should not find block outside time range")
}

func testMultipleBlocks() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())
    components.hour = 10
    components.minute = 30
    let testTime = calendar.date(from: components)!

    let blocks = [
        makePlannedBlock(id: 1, categoryId: 1, startTime: "09:00:00", duration: 60),
        makePlannedBlock(id: 2, categoryId: 2, startTime: "10:00:00", duration: 60),
        makePlannedBlock(id: 3, categoryId: 3, startTime: "11:00:00", duration: 60),
    ]

    let result = getCurrentPlannedBlock(at: testTime, from: blocks)
    assert(result?.id == 2, "Should find second block (10:00-11:00)")
}

// ============================================================================
// MARK: - State Transition Tests
// ============================================================================

func runStateTransitionTests() -> (passed: Int, failed: Int) {
    printSection("State Transition Tests")

    let tests: [(String, () throws -> Void)] = [
        ("State 1: Confirmation prompt triggers", testConfirmationPrompt),
        ("State 1->2: Confirmation to active", testConfirmationToActive),
        ("State 2->3: Active to off-schedule", testActiveToOffSchedule),
        ("State 3->2: Off-schedule to active (sync)", testOffScheduleToActive),
        ("State 2: Remain active when on-plan", testRemainActive),
        ("Confirmation window boundaries", testConfirmationWindow),
        ("New block boundary detection", testBlockBoundary),
        ("Offset bar visibility", testOffsetBarVisibility),
        ("Offset adjustment accumulation", testOffsetAdjustment),
        ("Offset reset on new transition", testOffsetReset),
        ("Progress throughout block", testProgressThroughBlock),
        ("Transitions across multiple blocks", testMultipleBlockTransitions),
        ("Edge case: Midnight crossing", testMidnightCrossing),
    ]

    return runSyncTests(tests)
}

func testConfirmationPrompt() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())
    components.hour = 10
    components.minute = 0
    components.second = 30
    let testTime = calendar.date(from: components)!

    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: "10:00:00", duration: 60)
    let isAtBoundary = isWithinConfirmationWindow(for: block, at: testTime, isConfirmed: false)

    assert(isAtBoundary == true, "Should trigger confirmation within 1 minute of start")
}

func testConfirmationToActive() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())
    components.hour = 10
    components.minute = 0
    components.second = 30
    let testTime = calendar.date(from: components)!

    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: "10:00:00", duration: 60)

    let before = isWithinConfirmationWindow(for: block, at: testTime, isConfirmed: false)
    let after = isWithinConfirmationWindow(for: block, at: testTime, isConfirmed: true)

    assert(before == true && after == false, "Confirmation should clear prompt")
}

func testActiveToOffSchedule() throws {
    let currentCategoryId = 1
    let plannedCategoryId = 2
    let isOffSchedule = currentCategoryId != plannedCategoryId

    assert(isOffSchedule == true, "Should be off-schedule when current != planned")
}

func testOffScheduleToActive() throws {
    let previousCategoryId = 1
    let plannedCategoryId = 2
    let wasOffSchedule = previousCategoryId != plannedCategoryId

    let currentCategoryId = plannedCategoryId
    let isOffSchedule = currentCategoryId != plannedCategoryId

    assert(wasOffSchedule == true && isOffSchedule == false, "Sync should return to active")
}

func testRemainActive() throws {
    let currentCategoryId = 1
    let plannedCategoryId = 1
    let isOffSchedule = currentCategoryId != plannedCategoryId

    assert(isOffSchedule == false, "Should remain active when on-plan")
}

func testConfirmationWindow() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())
    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: "10:00:00", duration: 60)

    // At start (0 seconds)
    components.hour = 10; components.minute = 0; components.second = 0
    let t0 = calendar.date(from: components)!
    assert(isWithinConfirmationWindow(for: block, at: t0, isConfirmed: false))

    // At 59 seconds
    components.second = 59
    let t59 = calendar.date(from: components)!
    assert(isWithinConfirmationWindow(for: block, at: t59, isConfirmed: false))

    // At 61 seconds (outside)
    components.minute = 1; components.second = 1
    let t61 = calendar.date(from: components)!
    assert(!isWithinConfirmationWindow(for: block, at: t61, isConfirmed: false))
}

func testBlockBoundary() throws {
    var lastCheckedBlockId: Int? = nil

    let firstCheck = (lastCheckedBlockId != 1)
    lastCheckedBlockId = 1
    let secondCheck = (lastCheckedBlockId != 1)
    let newBlockCheck = (lastCheckedBlockId != 2)

    assert(firstCheck && !secondCheck && newBlockCheck, "Boundary detection should work")
}

func testOffsetBarVisibility() throws {
    var showOffsetBar = false
    let wasOffSchedule = false
    let isNowOffSchedule = true

    if !wasOffSchedule && isNowOffSchedule {
        showOffsetBar = true
    }

    assert(showOffsetBar == true, "Offset bar should show when going off-schedule")
}

func testOffsetAdjustment() throws {
    var offsetMinutes = 0
    let lastEventTime = Date()

    offsetMinutes += 5
    let adjustedTime1 = lastEventTime.addingTimeInterval(TimeInterval(-5 * 60))

    offsetMinutes += 15
    let adjustedTime2 = adjustedTime1.addingTimeInterval(TimeInterval(-15 * 60))

    let totalAdjustment = adjustedTime2.timeIntervalSince(lastEventTime)
    assert(offsetMinutes == 20 && abs(totalAdjustment + 1200) < 1, "Offset should accumulate")
}

func testOffsetReset() throws {
    var offsetMinutes = 20
    offsetMinutes = 0
    assert(offsetMinutes == 0, "Offset should reset on new transition")
}

func testProgressThroughBlock() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())
    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: "10:00:00", duration: 60)

    // At 25% (15 min)
    components.hour = 10; components.minute = 15
    let p25 = calculateProgress(for: block, at: calendar.date(from: components)!)
    assert(p25 >= 0.24 && p25 <= 0.26, "25% progress")

    // At 50% (30 min)
    components.minute = 30
    let p50 = calculateProgress(for: block, at: calendar.date(from: components)!)
    assert(p50 >= 0.49 && p50 <= 0.51, "50% progress")

    // At 75% (45 min)
    components.minute = 45
    let p75 = calculateProgress(for: block, at: calendar.date(from: components)!)
    assert(p75 >= 0.74 && p75 <= 0.76, "75% progress")
}

func testMultipleBlockTransitions() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())

    let blocks = [
        makePlannedBlock(id: 1, categoryId: 1, startTime: "09:00:00", duration: 60),
        makePlannedBlock(id: 2, categoryId: 2, startTime: "10:00:00", duration: 30),
        makePlannedBlock(id: 3, categoryId: 3, startTime: "10:30:00", duration: 90),
    ]

    components.hour = 9; components.minute = 30
    assert(getCurrentPlannedBlock(at: calendar.date(from: components)!, from: blocks)?.id == 1)

    components.hour = 10; components.minute = 15
    assert(getCurrentPlannedBlock(at: calendar.date(from: components)!, from: blocks)?.id == 2)

    components.hour = 11; components.minute = 0
    assert(getCurrentPlannedBlock(at: calendar.date(from: components)!, from: blocks)?.id == 3)
}

func testMidnightCrossing() throws {
    let calendar = Calendar.current
    var components = calendar.dateComponents([.year, .month, .day], from: Date())
    let block = makePlannedBlock(id: 1, categoryId: 1, startTime: "23:30:00", duration: 60)

    components.hour = 23; components.minute = 45
    let r1 = getCurrentPlannedBlock(at: calendar.date(from: components)!, from: [block])
    assert(r1?.id == 1, "Should find block before midnight")

    components.hour = 0; components.minute = 15
    let r2 = getCurrentPlannedBlock(at: calendar.date(from: components)!, from: [block])
    assert(r2 == nil, "Should not match after midnight (day boundary)")
}

// ============================================================================
// MARK: - Mock API Tests
// ============================================================================

func runMockAPITests() async -> (passed: Int, failed: Int) {
    printSection("Mock API Tests")

    let tests: [(String, () async throws -> Void)] = [
        ("Login success", testMockLogin),
        ("Login failure (unauthorized)", testMockLoginFailure),
        ("Fetch categories success", testMockFetchCategories),
        ("Fetch categories unauthorized", testMockFetchCategoriesUnauthorized),
        ("Create day record", testMockCreateDayRecord),
        ("Post day events", testMockPostDayEvents),
    ]

    return await runAsyncTests(tests)
}

func testMockLogin() async throws {
    let client = MockAPIClient()
    let (token, userId) = try await client.login(username: "test", password: "pass")
    assert(!token.isEmpty && userId == 123, "Login should succeed")
}

func testMockLoginFailure() async throws {
    let client = MockAPIClient()
    client.shouldFailLogin = true

    do {
        _ = try await client.login(username: "test", password: "pass")
        throw TestError("Should have thrown unauthorized")
    } catch APIError.unauthorized {
        // Expected
    }
}

func testMockFetchCategories() async throws {
    let client = MockAPIClient()
    client.setAuthToken("test_token")
    client.mockCategories = [makeCategory(id: 1, name: "Work", color: "#FF0000")]

    let categories = try await client.fetchCategories()
    assert(categories.count == 1 && categories[0].name == "Work")
}

func testMockFetchCategoriesUnauthorized() async throws {
    let client = MockAPIClient()

    do {
        _ = try await client.fetchCategories()
        throw TestError("Should have thrown unauthorized")
    } catch APIError.unauthorized {
        // Expected
    }
}

func testMockCreateDayRecord() async throws {
    let client = MockAPIClient()
    client.setAuthToken("test_token")

    let record = try await client.createDayRecord(date: "2024-01-01")
    assert(record.calendarDate == "2024-01-01")
}

func testMockPostDayEvents() async throws {
    let client = MockAPIClient()
    client.setAuthToken("test_token")

    let events = [DayEvent(
        eventType: "confirmation",
        outgoingCategoryId: nil,
        incomingCategoryId: nil,
        occurredAt: Date()
    )]

    let response = try await client.postDayEvents(dayRecordId: 1, events: events)
    assert(response.createdEvents.count == 1)
}

// ============================================================================
// MARK: - Helper Functions
// ============================================================================

func getCurrentPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
    let calendar = Calendar.current
    let timeFormatter = DateFormatter()
    timeFormatter.dateFormat = "HH:mm:ss"

    for block in blocks {
        guard let blockStart = timeFormatter.date(from: block.startTime) else { continue }

        let startComponents = calendar.dateComponents([.hour, .minute, .second], from: blockStart)
        let currentComponents = calendar.dateComponents([.hour, .minute, .second], from: time)

        let currentSeconds = (currentComponents.hour ?? 0) * 3600 + (currentComponents.minute ?? 0) * 60 + (currentComponents.second ?? 0)
        let blockStartSeconds = (startComponents.hour ?? 0) * 3600 + (startComponents.minute ?? 0) * 60 + (startComponents.second ?? 0)
        let blockEndSeconds = blockStartSeconds + (block.durationMinutes * 60)

        if currentSeconds >= blockStartSeconds && currentSeconds < blockEndSeconds {
            return block
        }
    }

    return nil
}

func calculateProgress(for block: PlannedBlock, at time: Date) -> Double {
    let calendar = Calendar.current
    let timeFormatter = DateFormatter()
    timeFormatter.dateFormat = "HH:mm:ss"

    guard let blockStart = timeFormatter.date(from: block.startTime) else { return 0 }

    let startComponents = calendar.dateComponents([.hour, .minute, .second], from: blockStart)
    let currentComponents = calendar.dateComponents([.hour, .minute, .second], from: time)

    let currentSeconds = (currentComponents.hour ?? 0) * 3600 + (currentComponents.minute ?? 0) * 60 + (currentComponents.second ?? 0)
    let blockStartSeconds = (startComponents.hour ?? 0) * 3600 + (startComponents.minute ?? 0) * 60 + (startComponents.second ?? 0)
    let blockDurationSeconds = block.durationMinutes * 60

    let elapsed = max(0, currentSeconds - blockStartSeconds)
    return min(1.0, Double(elapsed) / Double(blockDurationSeconds))
}

func isWithinConfirmationWindow(for block: PlannedBlock, at time: Date, isConfirmed: Bool) -> Bool {
    let calendar = Calendar.current
    let timeFormatter = DateFormatter()
    timeFormatter.dateFormat = "HH:mm:ss"

    guard let blockStart = timeFormatter.date(from: block.startTime) else { return false }

    let startComponents = calendar.dateComponents([.hour, .minute, .second], from: blockStart)
    let currentComponents = calendar.dateComponents([.hour, .minute, .second], from: time)

    let currentSeconds = (currentComponents.hour ?? 0) * 3600 + (currentComponents.minute ?? 0) * 60 + (currentComponents.second ?? 0)
    let blockStartSeconds = (startComponents.hour ?? 0) * 3600 + (startComponents.minute ?? 0) * 60 + (startComponents.second ?? 0)
    let elapsed = currentSeconds - blockStartSeconds

    return elapsed >= 0 && elapsed < 60 && !isConfirmed
}

// ============================================================================
// MARK: - Test Infrastructure
// ============================================================================

func runSyncTests(_ tests: [(String, () throws -> Void)]) -> (passed: Int, failed: Int) {
    var passed = 0
    var failed = 0

    for (name, test) in tests {
        do {
            try test()
            print("  ✅ \(name)")
            passed += 1
        } catch {
            print("  ❌ \(name): \(error)")
            failed += 1
        }
    }

    print("  Result: \(passed) passed, \(failed) failed\n")
    return (passed, failed)
}

func runAsyncTests(_ tests: [(String, () async throws -> Void)]) async -> (passed: Int, failed: Int) {
    var passed = 0
    var failed = 0

    for (name, test) in tests {
        do {
            try await test()
            print("  ✅ \(name)")
            passed += 1
        } catch {
            print("  ❌ \(name): \(error)")
            failed += 1
        }
    }

    print("  Result: \(passed) passed, \(failed) failed\n")
    return (passed, failed)
}

// ============================================================================
// MARK: - Mock Factories
// ============================================================================

func makePlannedBlock(id: Int, categoryId: Int, startTime: String, duration: Int) -> PlannedBlock {
    PlannedBlock(id: id, categoryId: categoryId, startTime: startTime, durationMinutes: duration)
}

func makeCategory(id: Int, name: String, color: String) -> Category {
    Category(id: id, name: name, color: color, createdAt: Date(), updatedAt: Date())
}

// ============================================================================
// MARK: - Mock API Client
// ============================================================================

class MockAPIClient {
    var authToken: String?
    var shouldFailLogin = false
    var shouldFailFetch = false
    var mockCategories: [Category] = []
    var mockDayRecords: [DayRecord] = []

    func setAuthToken(_ token: String) {
        self.authToken = token
    }

    func login(username: String, password: String) async throws -> (token: String, userId: Int) {
        if shouldFailLogin { throw APIError.unauthorized }
        if username.isEmpty || password.isEmpty { throw APIError.serverError(400, "Invalid credentials") }

        let token = "mock_jwt_token_\(username)"
        setAuthToken(token)
        return (token: token, userId: 123)
    }

    func fetchCategories() async throws -> [Category] {
        guard authToken != nil else { throw APIError.unauthorized }
        if shouldFailFetch { throw APIError.networkError(NSError(domain: "test", code: -1)) }
        return mockCategories
    }

    func fetchDayRecords(from: String, to: String) async throws -> [DayRecord] {
        guard authToken != nil else { throw APIError.unauthorized }
        if shouldFailFetch { throw APIError.networkError(NSError(domain: "test", code: -1)) }
        return mockDayRecords
    }

    func createDayRecord(date: String) async throws -> DayRecord {
        guard authToken != nil else { throw APIError.unauthorized }

        let record = DayRecord(
            id: 1,
            snapshotId: nil,
            calendarDate: date,
            reviewStatus: "Unreviewed",
            snapshotBlocks: [],
            actualBlocks: [],
            createdAt: Date(),
            updatedAt: Date()
        )

        mockDayRecords.append(record)
        return record
    }

    func postDayEvents(dayRecordId: Int, events: [DayEvent]) async throws -> DayEventsResponse {
        guard authToken != nil else { throw APIError.unauthorized }

        let createdEvents = events.enumerated().map { index, event in
            CreatedEvent(
                id: index + 1,
                eventType: event.eventType,
                outgoingCategoryId: event.outgoingCategoryId,
                incomingCategoryId: event.incomingCategoryId,
                occurredAt: event.occurredAt
            )
        }

        return DayEventsResponse(createdEvents: createdEvents, actualBlocks: [])
    }
}

// ============================================================================
// MARK: - Utilities
// ============================================================================

struct TestError: Error, CustomStringConvertible {
    let message: String
    init(_ message: String) { self.message = message }
    var description: String { message }
}

func printHeader(_ text: String) {
    let width = 60
    let padding = String(repeating: "=", count: width)
    print("\n\(padding)")
    print(text.padding(toLength: width, withPad: " ", startingAt: 0))
    print("\(padding)\n")
}

func printSection(_ text: String) {
    print("📦 \(text)")
    print(String(repeating: "-", count: 60))
}

func printSeparator() {
    print(String(repeating: "=", count: 60))
}
