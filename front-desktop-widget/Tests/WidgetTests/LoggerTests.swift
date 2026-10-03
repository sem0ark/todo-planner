import Foundation

struct WidgetLoggerTests {
  func test_contextIsSortedAndFormatted() throws {
    let formatted = WidgetLogger.format(
      "[DEBUG] Cache loaded",
      context: ["usedCache": "true", "calendarDate": "2026-10-03"])

    try assertEqual(
      formatted,
      "[DEBUG] Cache loaded | calendarDate=2026-10-03 usedCache=true")
  }

  func test_emptyContextDoesNotAddSeparator() throws {
    try assertEqual(WidgetLogger.format("[ERROR] Cache failed"), "[ERROR] Cache failed")
  }
  static func testMethods() -> [TestCase] {
    let tests = WidgetLoggerTests()
    return [
      ("test_contextIsSortedAndFormatted", { try tests.test_contextIsSortedAndFormatted() }),
      ("test_emptyContextDoesNotAddSeparator", { try tests.test_emptyContextDoesNotAddSeparator() }),
    ]
  }
}
