# Testing Guide

## Running Tests

Run tests via Makefile in the repo root.

```bash
make test
```

## Test Organization

Tests are organized by component:

- **`test_helpers.go`** - Shared test utilities and database setup
- **`repository_*_test.go`** - Repository/data layer tests
- **`handler_*_test.go`** - HTTP handler tests

## Writing New Tests

All tests follow the **AAA pattern** (Arrange-Act-Assert):

```go
func TestFeatureName_Scenario(t *testing.T) {
    // Arrange - Set up test data and preconditions
    db := setupTestDB(t)
    repo := NewRepository(db)
    // ... setup

    // Act - Execute the code under test
    result, err := repo.DoSomething(ctx, input)

    // Assert - Verify the expected outcome
    if err != nil {
        t.Fatalf("DoSomething failed: %v", err)
    }
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

See [ARCHITECTURE.md](./ARCHITECTURE.md#testing-strategy) for detailed testing guidelines.

## Troubleshooting

**Tests are skipped:**
- Ensure `TEST_DATABASE_URL` is set
- Verify database is accessible

**Connection errors:**
- Check PostgreSQL is running
- Verify connection string format
- Ensure database exists

**Migration errors:**
- Database should be empty or cleaned between runs
- Check migration scripts in `migrations.go`
