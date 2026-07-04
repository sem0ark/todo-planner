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
- **`...._repository.go`** - Data access layer (e.g., `user_repository.go`). Wraps SQL and entity-related operations.
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

## Security Model

### Authentication
- **JWT**: Stateless tokens with HMAC-SHA256 signature
- **Password**: bcrypt hashing (cost 10)
- **User Isolation**: All queries filtered by authenticated user ID

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

1. **Create repository**: `tag_repository.go`
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
