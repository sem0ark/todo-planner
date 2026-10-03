# Testing Guide

## Running Tests

Run the widget tests from `front-desktop-widget/` through the Makefile:

```bash
make test
```

The `test` target generates the required configuration, compiles the standalone
Swift test runner, and executes it. The tests do not use XCTest or an Xcode test
target.

## Test Organization

Tests are organized by production responsibility:

- **`TestSupport.swift`** - Shared assertions, fixtures, mocks, and test harnesses
- **`PersistenceTests.swift`** - Bootstrap cache and local event store behavior
- **`RepositorySyncTests.swift`** - Authentication and remote synchronization
- **`ModelsAndInitializationTests.swift`** - Model encoding/decoding and widget initialization
- **`WidgetInteractionTests.swift`** - Widget state actions, transitions, offsets, confirmations, and failures
- **`ContentViewTests.swift`** - Authentication-driven content view mode decisions
- **`LoggerTests.swift`** - Logger formatting and context ordering
- **`TestRunner.swift`** - Standalone entry point, suite aggregation, and reporting

The Makefile discovers every Swift file under `Tests/WidgetTests/`, sorts them,
and compiles them together with production sources excluding `Sources/main.swift`
and `Sources/Views/`. New test files are therefore compiled automatically.

## Writing New Tests

Tests use the existing lightweight assertion helpers and should follow the
**AAA pattern** (Arrange-Act-Assert):

```swift
func test_featureName_scenario() async throws {
  // Arrange - Set up fixtures and preconditions.
  let harness = WidgetTestHarness(existingRecord: Fixtures.record())
  await harness.initialize()

  // Act - Execute the behavior under test.
  await harness.store.dispatch(.selectCategory(Fixtures.categoryA))

  // Assert - Verify the expected outcome.
  try assertEqual(harness.store.currentCategory?.id, Fixtures.categoryA.id)
}
```

When adding a test:

1. Put it in the file matching the production responsibility.
2. Reuse fixtures and mocks from `TestSupport.swift` where possible.
3. Add its `(name, function)` entry to that suite's `static testMethods()` method.
4. Run `make test` and verify the complete suite passes.

Keep test bodies focused on one behavior. Use table-driven or parameterized
approaches when several inputs exercise the same rule, and cover empty,
missing, boundary, and failure cases where applicable.

## Troubleshooting

**Swift compiler or SDK errors:**

- Ensure Swift is installed and `swiftc` is available on `PATH`.
- Verify Xcode command-line tools are installed with `xcode-select -p`.
- Confirm the selected SDK is available through `xcrun --show-sdk-path`.

**Tests fail after adding a test:**

- Confirm the test is registered exactly once in its suite's `testMethods()` method.
- Check that the test method name and owning test type match the registration.
- Review the first failing test; later failures may be consequences of shared
  state or an earlier failed setup.
