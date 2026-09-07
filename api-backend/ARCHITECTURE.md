# Backend Architecture

The API is a small Go HTTP service built around the standard library, `chi`, `pgx/v5`, and PostgreSQL. All files currently belong to the `main` package; the file names provide the logical separation between HTTP handlers, repositories, services, routing, and shared types.

## Request Flow

```text
HTTP request
    |
    v
router_*.go       route and method selection
    |
    v
auth.go           authentication middleware for protected routes
    |
    v
handler_*.go      decode, validate, call application code, encode response
    |
    +--> repository_*.go -> SQL and PostgreSQL mapping
                               |
                               v
                           PostgreSQL
```

Handlers own HTTP concerns. Repositories own persistence. Services coordinate domain operations that span repositories or require more than CRUD. Keep SQL out of handlers and keep HTTP types out of repository methods where practical.

## File Organization

Use the existing prefixes and place a new file beside the code it extends:

| File pattern | Responsibility |
| --- | --- |
| `main.go` | Configuration, database pool, migrations, and server startup |
| `handlers.go` | `API` composition and shared HTTP middleware |
| `handler_<feature>.go` | Feature request/response types, validation, and handlers |
| `router_<feature>.go` | Feature route registration and path parameter parsing |
| `repository_<resource>.go` | Persistence models, SQL queries, and repository methods |
| `shared_*.go` | Cross-feature infrastructure and shared types |
| `*_test.go` | Focused handler, repository, or domain tests |
| `migrations.go` | Versioned PostgreSQL schema changes |


## Naming and Type Conventions

Follow standard Go naming while using descriptive domain names:

- Use `PascalCase` for exported types, functions, fields, and constants; use `camelCase` for unexported identifiers.
- Use descriptive names for domain concepts: `userID`, `categoryID`, `parsedDate`, `databaseError`. For types already explicit from context, idiomatic Go abbreviations are acceptable.
- Use one domain term consistently. For example, use `category` rather than mixing `category`, `cat`, and `blockCategory` for the same local value.
- Return errors explicitly. Use early returns to keep the successful path flat.
- Keep constructors named `New<Type>` and return pointers for repositories and services that hold shared dependencies.
- Use `time.Time` for database values and `APITimestamp` or `APIScheduleTime` only at the JSON boundary. `shared_types_time.go` is the single source of truth for public date and time formats.

Example of a repository constructor and method:

```go
type CategoryRepository struct {
    db *pgxpool.Pool
}

func NewCategoryRepository(database *pgxpool.Pool) *CategoryRepository {
    return &CategoryRepository{db: database}
}

func (repository *CategoryRepository) FindByUser(
    context context.Context,
    userID int,
) ([]BlockCategory, error) {
    // Query, scan, and return rows. Do not write HTTP responses here.
}
```

## API Composition

Construct dependencies once in `NewAPI`. Keep dependency wiring visible and avoid package-level mutable state:

```go
type API struct {
    db                *pgxpool.Pool
    jwtSecret         string
    logger            *Logger
    userRepo          *UserRepository
    categoryRepo      *CategoryRepository
    dayRecordRepo     *DayRecordRepository
    dayService        *DayService
}

func NewAPI(database *pgxpool.Pool, jwtSecret string, logger *Logger) *API {
    api := &API{
        db:            database,
        jwtSecret:     jwtSecret,
        logger:        logger,
        userRepo:      NewUserRepository(database),
        categoryRepo:  NewCategoryRepository(database),
        dayRecordRepo: NewDayRecordRepository(database),
    }
    api.dayService = NewDayService(api.dayRecordRepo, api.categoryRepo)
    return api
}
```

When adding a dependency, add its field and constructor call here. Do not create repositories inside individual handlers.

## Handler Responsibilities

A handler should authenticate through the route middleware, decode input, validate request-specific rules, call a repository or service, and write one response. Use `HTTPError` for internal failures so logs retain diagnostic details while clients receive a safe public message.

```go
func (api *API) getCategoriesHandler(w http.ResponseWriter, r *http.Request) {
    userID := userIDFromRequest(r)

    categories, err := api.categoryRepo.FindByUser(r.Context(), userID)
    if err != nil {
        HTTPError(
            w,
            r,
            api.logger,
            http.StatusInternalServerError,
            "failed to fetch categories",
            err,
            map[string]interface{}{"user_id": userID},
        )
        return
    }

    writeJSON(CategoriesResponse{
        Categories: categories,
    })
}
```

Use `AppError` from `shared_types_error.go` for expected application failures. It carries the public message and HTTP status while preserving an optional cause for `errors.Is` and internal logging. Handlers should call `writeAppError` before falling back to `HTTPError` for unexpected failures.

Return `400` for malformed input, `401` for missing authentication, `404` for missing resources, and `500` for unexpected persistence or infrastructure failures. Do not expose SQL errors, tokens, password hashes, or stack traces in HTTP responses.

Request and response structs belong in the owning handler file. Use explicit JSON tags. To ensure collection fields marshal to `[]` instead of `null`, initialize them in your DTO mapping function rather than in every handler:

```go
func toCategoriesResponse(models []Category) CategoriesResponse {
    res := make([]PublicCategory, 0, len(models)) // Initializing ensures [] not null
    for _, m := range models {
        res = append(res, toPublicCategory(m))
    }
    return CategoriesResponse{Categories: res}
}
```

There is intentionally no shared `models.go` file. Persistence models belong in their owning `repository_<resource>.go` file, while request, response, and validation types belong in the relevant `handler_<feature>.go` file. Move a type only when its ownership changes; do not recreate a general-purpose model package for convenience.

## Routing and Middleware

Keep route registration in `router_*.go`. Parse path parameters at the route boundary and pass typed values to handlers when that is the established pattern. Protected routes use `protectedHandler`, which applies the existing authentication middleware:

```go
router.HandleFunc(
    "/categories",
    api.protectedHandler(api.getCategoriesHandler),
)
```

The middleware order is:

```text
logging -> CORS -> authentication -> handler
```

Authentication should be applied once at the route boundary. Endpoint handlers should still use `getUserID` as a defensive check before accessing user-scoped data.

## Repositories and PostgreSQL

Repositories use `pgxpool.Pool` and parameterized SQL. Every user-scoped query must constrain by the authenticated `userID`; never rely on the handler alone for data isolation. **Crucially, pass `context.Context` from the request to all database calls** so that if a client hangs up, the query is cancelled immediately rather than consuming resources.

```go
func (repository *CategoryRepository) Delete(
    ctx context.Context,
    categoryID int,
    userID int,
) error {
    commandTag, err := repository.db.Exec(ctx, `
        UPDATE block_categories
        SET is_deleted = TRUE, updated_at = $3
        WHERE id = $1 AND user_id = $2
    `, categoryID, userID, time.Now().UTC())
    if err != nil {
        return err
    }
    if commandTag.RowsAffected() == 0 {
        return ErrCategoryNotFound
    }
    return nil
}
```

Use `QueryRow` for one row, `Query` with `defer rows.Close()` for collections, and always return `rows.Err()` after iteration. Keep SQL columns and scan destinations in the same order. Use transactions for multi-statement operations and pass the transaction through the repository operation. Always use `rows.Err()` to catch iteration errors, not just `error` from `Scan()`.

Repository persistence models may use `time.Time` and PostgreSQL-native values directly.

Convert values only when mapping to a public JSON DTO. Use converter functions to centralize the mapping logic, especially for collections:
```go
type PublicRecord struct {
    CalendarDate string       `json:"calendar_date"`
    CreatedAt    APITimestamp `json:"created_at"`
}

func toPublicRecord(record DayRecord) PublicRecord {
    return PublicRecord{
        CalendarDate: record.CalendarDate,
        CreatedAt:    APITimestamp(record.CreatedAt),
    }
}

func toPublicRecords(records []DayRecord) []PublicRecord {
    result := make([]PublicRecord, 0, len(records))
    for _, record := range records {
        result = append(result, toPublicRecord(record))
    }
    return result
}
```

## Domain Services

Use a service when an operation coordinates multiple repositories, performs a
workflow, or applies domain rules that do not belong to HTTP or SQL. Keep
simple CRUD operations in the relevant repository.

```go
type DayService struct {
    dayRecordRepository *DayRecordRepository
    categoryRepository  *CategoryRepository
}

func NewDayService(
    dayRecordRepository *DayRecordRepository,
    categoryRepository *CategoryRepository,
) *DayService {
    return &DayService{
        dayRecordRepository: dayRecordRepository,
        categoryRepository:  categoryRepository,
    }
}
```

Services return domain values and errors. They do not write HTTP responses, inspect headers, or format JSON.

## Error Handling and Logging

Use sentinel or typed errors for expected domain conditions, and wrap lower level errors with context when the caller benefits from it:

```go
if error != nil {
    return fmt.Errorf("load categories for user %d: %w", userID, error)
}
```

At the HTTP boundary, classify expected errors into public status codes and send unexpected errors through `HTTPError`. Structured logs may include safe identifiers such as `user_id` and `category_id`, but must not include:

- passwords, password hashes, JWTs, authorization headers, or secrets;
- request bodies containing sensitive credentials;
- raw database errors in the response body.

Use `Logger.Info`, `Logger.Warn`, and `Logger.Error` for application events. The logger writes structured JSON and includes stack traces for error-level entries.

## Adding a Feature or Table

For a new resource such as `tags`:

1. Add the schema change in `migrations.go` using the existing migration conventions.
2. Add `repository_tags.go` with `Tag`, `TagInput`, `TagRepository`, and a `NewTagRepository` constructor.
3. Add `tagRepo *TagRepository` and its constructor call in `API` and `NewAPI`.
4. Add `handler_tags.go` for request/response types, validation, and handlers.
   - **Syntactic Validation** (e.g., email format, required fields) belongs in the handler.
   - **Semantic/Domain Validation** (e.g., does this tag belong to this user, are the dependencies valid?) belongs in the repository or service.
5. Add `router_tags.go` and register public or protected routes consistently.
6. Add repository integration tests and handler tests for success, validation, authorization, not-found, and database-error paths.

Keep the change scoped to the feature. Do not introduce a new framework, ORM, package hierarchy, or abstraction layer for a single endpoint.

## Resource and Performance Guidelines

The service runs with a deliberately small connection pool. Configure pool limits in startup code and reuse the pool for the lifetime of the process:

```go
config.MaxConns = 5
config.MinConns = 0
config.MaxConnIdleTime = 2 * time.Minute
```

Preallocate slices only when you know the exact size or can count it efficiently:

```go
categories := make([]BlockCategory, 0, len(rows)) // Use the actual row count
```

Avoid arbitrary preallocation like `make([]T, 0, 10)`. If you don't know the size, use `var categories []BlockCategory` and let `append` handle growth; it is efficient and idiomatic.

Prefer straightforward SQL and bounded queries over speculative caching.
Measure before optimizing; correctness, cancellation, and user isolation are
more important than small allocation reductions.

## Testing Strategy

Tests use the Arrange-Act-Assert structure and descriptive names. Repository tests are integration tests against the PostgreSQL test database. Handler tests use `httptest` and verify status codes, JSON, validation, and error handling. Domain logic tests should avoid a database when dependencies can be isolated naturally.

Each feature should cover:

1. the successful operation;
2. malformed and boundary input;
3. missing authentication and cross-user access;
4. missing resources and database errors;
5. empty collections, null values, and relevant state transitions.

Use table-driven subtests for related cases:

```go
func TestParseBlockStartTime(t *testing.T) {
    testCases := []struct {
        name      string
        input     string
        shouldErr bool
    }{
        {name: "seconds", input: "08:30:00"},
        {name: "minutes", input: "08:30"},
        {name: "invalid", input: "not-a-time", shouldErr: true},
    }

    for _, testCase := range testCases {
        t.Run(testCase.name, func(t *testing.T) {
            _, err := parseBlockStartTime(testCase.input)
            if (err != nil) != testCase.shouldErr {
                t.Fatalf("unexpected error state: %v", err)
            }
        })
    }
}
```

For more details, see [TESTING](./TESTING.md)
