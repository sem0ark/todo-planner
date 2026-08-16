# Backend Architecture

## Lightweight Repository Pattern

The backend uses a clean, lightweight repository pattern (service-per-table) to keep code organized without the bloat of a full ORM.

### Structure

```
┌──────────────┐
│   Handlers   │  <- HTTP layer (routing, validation)
└──────┬───────┘
       │
┌──────▼────────┐
│ Repositories  │  <- Data access layer (CRUD)
└──────┬────────┘
       │
┌──────▼────────┐
│   Database    │  <- PostgreSQL
└───────────────┘
```

### Files

- **`handlers.go`** - Main API structure, shared middleware, and request/response utilities.
- **`handler_....go`** - Feature-specific HTTP handlers (e.g., `handler_health.go`). Define routes, validation, and JSON conversion; delegate business logic to repositories.
- **`repository_....go`** - Data access layer (e.g., `repository_users.go`). Wraps SQL and entity-related operations.
- **`logger.go`** - Structured JSON logging system with request/error tracking. See [LOGGING.md](LOGGING.md) for details.
- **`models.go`** - Shared data structures and entities.
- **`main.go`** - Entry point, environment configuration, and server initialization.

### Benefits

1. **Separation of Concerns**
   - Handlers: HTTP logic only (request/response, validation)
   - Repositories: Data access and business logic

2. **Clean Code**
   - No SQL in handlers
   - Easy to test each layer independently

3. **One Service = One Table**
   - Each repository provides basic CRUD operations

## Resource Optimization for .25 CPU Container

### Pre-allocated Slices
**Benefit**: Reduce memory allocations and GC pressure in performance-critical sections.

### Connection Pooling
```go
// main.go
config.MaxConns = 5
config.MinConns = 0
config.MaxConnIdleTime = 2 * time.Minute
```
**Benefit**: Efficient connection reuse, low memory footprint,

## Middleware Chain

All HTTP requests flow through a centralized middleware stack:

```
Request -> LoggingMiddleware -> CORSMiddleware -> AuthMiddleware -> Handler
```

1. **LoggingMiddleware** - Logs all requests/responses with timing, captures panics, adds stack traces for errors
2. **CORSMiddleware** - Handles CORS headers and preflight requests based on `CORS_ALLOWED_ORIGINS`
3. **AuthMiddleware** - Validates JWT tokens, extracts user context (only on protected routes)
4. **Handler** - Business logic and response generation

### Error Handling

Use `HTTPError()` to log detailed internal errors while returning generic messages to users:

```go
HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to create user", err, map[string]interface{}{
    "username": req.Username,
})
```

This logs the full error with stack trace internally but returns just "failed to create user" to the client.

## Security Model

### Authentication
- **JWT**: Stateless tokens with HMAC-SHA256 signature
- **Password**: bcrypt hashing (cost 10)
- **User Isolation**: All queries filtered by authenticated user ID

### Logging Security
- **No PII in logs**: Tokens, passwords, and authorization headers are never logged
- **Failed auth attempts**: Logged at WARN level with username and IP (for monitoring)
- **Successful logins**: Logged at INFO level with user ID and IP

## Repository Pattern Example

### TodoRepository API
```go
type TodoRepository struct {
    db            *pgxpool.Pool
    encryptionKey []byte
}

// Clean, simple API
func (r *TodoRepository) FindByUser(ctx context.Context, userID int) ([]Todo, error)
func (r *TodoRepository) FindByID(ctx context.Context, id, userID int) (*Todo, error)
func (r *TodoRepository) Create(ctx context.Context, input TodoInput, userID int) (*Todo, error)
func (r *TodoRepository) Update(ctx context.Context, id int, input TodoInput, userID int) (*Todo, error)
func (r *TodoRepository) Delete(ctx context.Context, id, userID int) error
```

### Handler Simplicity
```go
// Before (60+ lines of encryption/DB logic)
func (api *API) getTodos(w http.ResponseWriter, r *http.Request) {
    // ... auth check
    // ... query builder
    // ... row scanning
    // ... encrypt/decrypt per field
    // ... error handling
}

// After (10 lines)
func (api *API) getTodos(w http.ResponseWriter, r *http.Request) {
    userID, _ := getUserID(r.Context())
    todos, err := api.todoRepo.FindByUser(r.Context(), userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(todos)
}
```

## Adding New Tables

To add a new table (e.g., `tags`):

1. **Create repository**: `repository_tags.go`
```go
type TagRepository struct {
    db *pgxpool.Pool
}

func (r *TagRepository) FindAll(ctx context.Context) ([]Tag, error) { ... }
func (r *TagRepository) Create(ctx context.Context, input TagInput) (*Tag, error) { ... }
// ... etc
```

2. **Add to API struct**: `handlers.go`
```go
type API struct {
    db       *pgxpool.Pool
    userRepo *UserRepository
    tagRepo  *TagRepository  // <- Add here
}
```

3. **Create handlers**: `handler_tags.go`
```go
func (api *API) getTags(w http.ResponseWriter, r *http.Request) {
    tags, err := api.tagRepo.FindAll(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(tags)
}
```

4. **Register routes**: `main.go`
```go
http.HandleFunc("/tags", api.getTags)
```

## Memory Profile (Estimated)

| Component | Memory | Notes |
|-----------|--------|-------|
| Go runtime | ~10 MB | Minimal overhead |
| pgx pool (5 conns) | ~5 MB | Lightweight driver |
| Cipher cache | <1 MB | One AES cipher per key |
| Request buffers | ~2 MB | JSON encoding/decoding |
| **Total baseline** | **~20 MB** | Well within .25 CPU limits |

Per request overhead: ~100-200 KB (JSON + crypto buffers)

## Performance Characteristics

- **Cold start**: ~50ms (Go binary + DB connection)
- **Typical response**: 10-50ms (depending on DB latency)
- **Encryption overhead**: 1-2ms per request
- **JWT verification**: <1ms
- **Memory per request**: 100-200 KB

## Trade-offs

### What We Have
- Clean code separation
- Transparent encryption
- Type-safe CRUD operations
- Minimal dependencies
- Low resource usage

### What We Don't Have (by design)
- Full ORM features (migrations, relations, query builder)
- Automatic schema generation
- Complex query DSL
- Code generation

**Rationale**: Keep it simple and explicit. SQL is readable, repositories are lightweight, and we avoid ORM complexity/overhead.

## Testing Strategy

### AAA Pattern (Arrange-Act-Assert)

All tests follow the AAA pattern for clarity and consistency:
1. **Arrange** - Set up test data and preconditions
2. **Act** - Execute the code under test
3. **Assert** - Verify the expected outcome

This pattern makes tests self-documenting and easy to understand at a glance.

### Test Types

#### Repository Tests
Test data access logic in isolation. These are integration tests that use a real test database.

**Example:**
```go
func TestUserSettingsRepository_Update(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    repo := NewUserSettingsRepository(db)
    user := createTestUser(t, db, "testuser", "password123")
    ctx := context.Background()
    repo.GetOrCreate(ctx, user.ID)

    // Act
    newTime := "06:30:00"
    updated, err := repo.Update(ctx, user.ID, newTime)

    // Assert
    if err != nil {
        t.Fatalf("Update failed: %v", err)
    }
    if updated.DayBoundaryTime != newTime {
        t.Errorf("Expected DayBoundaryTime '%s', got '%s'", newTime, updated.DayBoundaryTime)
    }
}
```

#### Handler Tests
Test HTTP layer behavior using `httptest`. Verify request validation, response formatting, status codes, and error handling.

**Example:**
```go
func TestDeleteAccountHandler_Success(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    api := NewAPI(db, "test-secret")
    user := createTestUser(t, db, "testuser", "password123")

    reqBody := DeleteAccountInput{Password: "password123"}
    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    ctx := withUserID(context.Background(), user.ID)
    req = req.WithContext(ctx)
    
    w := httptest.NewRecorder()

    // Act
    api.deleteAccountHandler(w, req)

    // Assert
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    var response DeleteAccountResponse
    json.NewDecoder(w.Body).Decode(&response)
    if !response.Deleted {
        t.Error("Expected deleted=true in response")
    }
}
```

### Test Helpers

Common setup utilities are centralized in `test_helpers.go`:

- **`setupTestDB(t)`** - Creates isolated test database with migrations
- **`cleanupTestDB(t, db)`** - Truncates all tables after tests
- **`createTestUser(t, db, username, password)`** - Creates a test user

### Running Tests

Tests require a test database. Set the connection string:

```bash
export TEST_DATABASE_URL="postgresql://user:pass@localhost:5432/test_db"
go test ./...
```

Tests without `TEST_DATABASE_URL` are automatically skipped.

### Test Coverage Guidelines

Each new feature should include:

1. **Happy path** - Normal successful operation
2. **Error cases** - Invalid input, missing data, unauthorized access
3. **Edge cases** - Boundary values, empty strings, null values
4. **Security** - Authentication, authorization, password verification
5. **Side effects** - Cascade deletes, state changes, idempotency

### Example Test Suite Structure

```go
// Repository: user_settings_repository_test.go
- TestUserSettingsRepository_GetOrCreate
- TestUserSettingsRepository_GetOrCreate_Idempotent
- TestUserSettingsRepository_Update
- TestUserSettingsRepository_Update_BeforeCreate
- TestUserSettingsRepository_MultipleUsers

// Handler: handler_settings_test.go
- TestGetSettingsHandler_Success
- TestGetSettingsHandler_NoAuth
- TestGetSettingsHandler_WrongMethod
- TestPutSettingsHandler_Success
- TestPutSettingsHandler_InvalidTimeFormat
- TestPutSettingsHandler_ValidTimeFormats
- TestPutSettingsHandler_NoAuth
- TestPutSettingsHandler_InvalidJSON
```

### Best Practices

1. **Isolation** - Each test is independent, cleanup between tests
2. **Clarity** - Test names describe what's being tested and expected outcome
3. **Comments** - AAA sections clearly marked with comments
4. **t.Helper()** - Mark helper functions to improve error reporting
5. **Subtests** - Use `t.Run()` for testing multiple similar cases
6. **Minimal mocking** - Use real database for integration tests, only mock external services
