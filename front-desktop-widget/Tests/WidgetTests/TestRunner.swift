import Foundation

struct TestResult {
  let name: String
  let passed: Bool
  let error: String?
}

@main
struct TestRunner {
  static func main() async {
    print("Running WidgetStateStore tests...")
    print("")

    let testMethods = WidgetLoggerTests.testMethods()
      + AuthControllerTests.testMethods()
      + PersistenceTests.testMethods()
      + RepositorySyncTests.testMethods()
      + ModelsAndInitializationTests.testMethods()
      + WidgetInteractionTests.testMethods()
    var results: [TestResult] = []

    for (name, testFn) in testMethods {
      do {
        try await testFn()
        results.append(TestResult(name: name, passed: true, error: nil))
        print("✓ \(name)")
      } catch let error as AssertionError {
        results.append(TestResult(name: name, passed: false, error: String(describing: error)))
        print("✗ \(name): \(error)")
      } catch {
        results.append(TestResult(name: name, passed: false, error: String(describing: error)))
        print("✗ \(name): \(error)")
      }
    }

    print("")
    let passed = results.filter(\.passed).count
    let total = results.count
    print("Passed: \(passed)/\(total)")

    if passed < total {
      exit(1)
    }
  }
}
