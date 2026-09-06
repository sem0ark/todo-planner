# General Coding Conventions (Python, Go, Swift)

## Function & API Design

- Write clear, concise comments and docstrings explaining the **"why"** and usage scenarios, "what" must be described briefly by the name only.
- Ensure functions have descriptive, intention-revealing names.
- Use explicit type systems / type annotations idiomatic to the language:
  - **Python**: Native built-in hints (e.g., `list[str]`, `dict[str, int]`).
  - **Go**: Explicit concrete types, interfaces, and generics where appropriate.
  - **Swift**: Strong types, generics, and protocol-oriented design.
- Avoid redundancy: do not repeat parameter/return types in doc comments if static types or hints already convey them.
- Break down complex functions into smaller, single-responsibility units.
- Avoid dynamic reflection or type inspection workarounds (e.g., Python `hasattr`/`getattr`, Go `reflect`, Swift runtime `Mirror`/unsafe casts) to bypass type safety; refactor leaky abstractions into explicit interfaces, protocols, or base types instead.

## General Instructions

- **Readability and clarity over brevity**: Write code that is easily readable and maintainable over compact or "clever" one-liners.
- **Explain algorithms**: Document the approach, time/space complexity, and non-obvious design choices.
- **Explicit error handling**: Handle edge cases and failures cleanly according to language idioms:
  - **Python**: Focused `try/except` blocks with specific exception types.
  - **Go**: Explicit `error` return values; avoid unchecked `nil` or unnecessary `panic`.
  - **Swift**: Typed `throws`, `Result`, or optional unwrapping (avoid force-unwrapping `!`).
- **Dependencies**: Explicitly document third-party libraries and their purpose in comments or module headers.
- **File structure**: Place all imports and package references at the top of the file. Avoid inline or nested imports/includes.

## Naming Conventions

### 1. No Single-Character Variables or Arbitrary Abbreviations
**Use full, unabbreviated words.** Never shorten names or use single-character variables for brevity. Focus on domain lexicon and clarity.

| Avoid / Anti-Pattern | Preferred Full Name |
| :--- | :--- |
| `resp` | `response` |
| `req` | `request` |
| `ctx` | `context` |
| `msg` | `message` |
| `tx`, `txn` | `transaction` |
| `param` | `parameter` |
| `arg` | `argument` |
| `prev` | `previous` |
| `cur`, `curr` | `current` |
| `idx`, `i` | `index` |
| `val`, `v` | `value` |
| `doc` | `document` |
| `func`, `fn` | `function` |
| `err`, `exc` | `error`, `exception` |
| `cmd` | `command` |
| `src` | `source` |
| `dst`, `dest` | `destination` |
| `num`, `n` | `number` / `count` |
| `res` | `result` |
| `len` | `length` |
| `ref` | `reference` |
| `mgr` | `manager` |
| `conn` | `connection` |
| `rec` | `record` |
| `cb` | `callback` |

### 2. Loop & Iteration Variables
Iterators, closures, and collection loops must use full, descriptive names:

- **Python**: `for order in orders:` *(not `for o in orders:`)*
- **Go**: `for rowIndex, row := range rows` *(not `for i, r := range rows`)*
- **Swift**: `for payment in pendingPayments` *(not `for p in pendingPayments`)*

### 3. Maintain a Consistent Domain Lexicon
- Establish a single canonical term per domain concept and use it uniformly across files, variable names, types, APIs, and documentation.
  - *Example*: Do not mix `customer`, `client`, `user`, and `buyer` for the same entity.
  - *Example*: Do not mix `fetchOrders`, `getOrders`, `retrieveOrders`, and `loadOrders` without intentional architectural differences.

### 4. Descriptive Compound Names
Prefer specific, compound identifiers over generic words to convey intent and scope:
- `retryDelaySeconds` / `retry_delay_seconds` instead of `delay`
- `maximumConnectionCount` / `maximum_connection_count` instead of `limit`
- `isPaymentOverdue` / `is_payment_overdue` instead of `overdue`

### 5. Casing Conventions per Language
Apply the unabbreviated lexicon using each language's standard casing style:

- **Python**: `snake_case` (functions, variables, modules), `PascalCase` (classes, exceptions).
- **Go**: `camelCase` (unexported types/fields/functions), `PascalCase` (exported types/fields/functions).
- **Swift**: `camelCase` (functions, properties, variables, enum cases), `PascalCase` (types, protocols).

## Code Style and Formatting

- Use the official formatter and linter for each ecosystem:
  - **Python**: PEP 8 standards (`ruff` or `flake8` / `black`).
  - **Go**: `gofmt` and `golangci-lint`.
  - **Swift**: Swift API Design Guidelines (`swift-format` or `SwiftLint`).
- Keep lines within reasonable lengths (typically 80–100 characters depending on project configuration).
- Place documentation comments directly above or at the beginning of the declaration (docstrings in Python, `//` in Go, `///` in Swift).
- Avoid deeply nested structures; prefer early returns / guard clauses to reduce nesting depth.

## Testing Guidelines

- **AAA Pattern**: Structure tests strictly around **Arrange-Act-Assert**.
- **Low Cyclomatic Complexity**: Strive for a cyclomatic complexity of **1** inside test bodies (linear flow, avoiding conditional branches and loops within the test body).
- **Table-Driven / Parameterized Testing**: Use declarative, data-driven tables for parameterized inputs rather than writing custom test loops:
  - **Python**: `pytest.mark.parametrize`
  - **Go**: Table-driven tests using `t.Run(testCase.name, ...)`
  - **Swift**: Swift Testing `@Test(arguments: ...)` or parameter arrays
- **Edge Cases**: Always include tests for boundary conditions: empty collections, nil/null inputs, zero values, large datasets, and error conditions.

## Code & Documentation Examples

### Python
```python
import math


def calculate_circle_area(radius: float) -> float:
    """Calculates the surface area of a circle for geometric computations."""
    if radius < 0:
        raise ValueError("Radius cannot be negative.")
    return math.pi * (radius ** 2)
```

### Go
```go
package geometry

import (
	"errors"
	"math"
)

// CalculateCircleArea calculates the surface area of a circle for geometric computations.
func CalculateCircleArea(radius float64) (float64, error) {
	if radius < 0 {
		return 0, errors.New("radius cannot be negative")
	}
	return math.Pi * math.Pow(radius, 2), nil
}
```

### Swift
```swift
import Foundation

enum GeometryError: Error {
    case negativeRadius
}

/// Calculates the surface area of a circle for geometric computations.
func calculateCircleArea(radius: Double) throws -> Double {
    guard radius >= 0 else {
        throw GeometryError.negativeRadius
    }
    return Double.pi * pow(radius, 2)
}
```
