# go-repository-cache

A **type-safe caching decorator** for [go-repository-bun](https://github.com/goliatone/go-repository-bun) repositories using [sturdyc](https://github.com/viccon/sturdyc) for stampede-safe caching.

## Features

- **Drop-in compatibility**: Implements the same `Repository[T]` interface
- **Stampede protection**: Non blocking reads with inflight request deduplication
- **Type-safe caching**: Generic implementation maintains full type safety
- **Tag based invalidation**: Registers read keys under tags and invalidates on writes
- **Selective caching**: Only read operations are cached; writes pass through
- **Transaction awareness**: Bypasses cache for transactional operations
- **Smart key generation**: Handles complex criteria and sanitizes namespaces for Redis/Memcache safety
- **Configurable**: Pluggable key serialization and cache configuration

## Quick Start

### Installation

```bash
go get github.com/goliatone/go-repository-cache
```

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/goliatone/go-repository-cache/cache"
    "github.com/goliatone/go-repository-cache/repositorycache"
    // Your existing repository
    "yourproject/repositories"
)

func main() {
    // Your existing repository
    baseRepo := repositories.NewUserRepository(db)

    // Configure cache
    cacheConfig := cache.Config{
        Capacity:             10_000,
        NumShards:            256,
        TTL:                  30 * time.Minute,
        EvictionPercentage:   10,
        MissingRecordStorage: true,
    }

    // Create cache components
    cacheService, err := cache.NewCacheService(cacheConfig)
    if err != nil {
        panic(err)
    }
    keySerializer := cache.NewDefaultKeySerializer()

    // Wrap with caching
    cachedRepo := repositorycache.New(baseRepo, cacheService, keySerializer)

    // Use exactly like your original repository
    ctx := context.Background()
    user, err := cachedRepo.GetByID(ctx, "user-123") // First call hits DB
    user, err = cachedRepo.GetByID(ctx, "user-123")  // Second call hits cache
}
```

Need to target additional identifier fields (for example, a `Slug` column)? Pass them explicitly:

```go
cachedRepo := repositorycache.NewWithIdentifierFields(
    baseRepo,
    cacheService,
    keySerializer,
    "Slug", "ExternalID",
)
```

### Using the Dependency Injection Container

```go
import (
    "time"

    "github.com/goliatone/go-repository-cache/cache"
    "github.com/goliatone/go-repository-cache/pkg/di"
)

func main() {
    // Configure cache
    config := cache.Config{
        TTL: 30 * time.Minute,
        // ... other options
    }

    // Create container
    container, err := di.NewContainer(config)
    if err != nil {
        panic(err)
    }

    // Your base repository
    baseRepo := repositories.NewUserRepository(db)

    // Get cached version
    cachedRepo := container.NewCachedRepository(baseRepo)

    // Use as normal
    users, total, err := cachedRepo.List(ctx, repository.Limit(10))
}
```

## How It Works

### Caching Strategy

**Cached Operations** (performance benefit):
- `Get`, `GetByID`, `GetByIdentifier`
- `List`, `Count`

**Pass-through Operations** (consistency guarantee):
- All write operations (`Create`, `Update`, `Delete`, etc.)
- All transaction methods (`*Tx` variants)
- Raw SQL queries

### Key Generation

The library automatically generates stable cache keys from:
- Method name (`GetByID`, `List`, etc.)
- All parameters including complex criteria
- Function pointers (stable within process lifetime)

```go
//these generate different cache keys:
repo.GetByID(ctx, "123")
repo.GetByID(ctx, "123", repository.WithDeleted())
repo.List(ctx, repository.Where("active", true))
repo.List(ctx, repository.Where("active", false))
```

### Scope Aware Keys

When your base repository uses the `go-repository-bun` scope system, the decorator automatically folds the active scope names and any `WithScopeData` payloads into every cached key. Tenant/session specific contexts therefore never share cached rows:

```go
ctx := repository.WithSelectScopes(ctx, "tenant")
ctx = repository.WithScopeData(ctx, "tenant", tenantID)

// Key includes both the "tenant" scope name and the concrete tenantID value
users, total, err := cachedRepo.List(ctx, repository.SelectPaginate(25, 0))
```

Scopes registered or defaulted through the cached repository are forwarded to the underlying repository:

```go
cachedRepo.RegisterScope("tenant", tenantScopeDefinition)
if err := cachedRepo.SetScopeDefaults(repository.ScopeDefaults{
    Select: []string{"tenant"},
}); err != nil {
    panic(err)
}
```

This keeps cache keys aligned with whatever filters the base repository enforces while respecting `WithoutDefaultScopes`, `WithSelectScopes`, and other helpers.

### Transaction Handling

Operations within transactions bypass the cache to ensure consistency:

```go
// This hits the cache
user, err := repo.GetByID(ctx, "123")

// This bypasses the cache and goes directly to DB
err = repo.WithTx(ctx, func(ctx context.Context, tx bun.IDB) error {
    user, err := repo.GetByIDTx(ctx, tx, "123") // Direct DB access
    return repo.UpdateTx(ctx, tx, updatedUser)
})
```

## Configuration

### Cache Configuration

```go
config := cache.Config{
    TTL:                30 * time.Minute,
    NumShards:          256,
    EvictionPercentage: 10,

    // background refresh before expiry
    EarlyRefresh: &cache.EarlyRefreshConfig{
        MinAsyncRefreshTime: 20 * time.Minute,
        MaxAsyncRefreshTime: 25 * time.Minute,
        SyncRefreshTime:     30 * time.Minute,
        RetryBaseDelay:      1 * time.Second,
    },

    // Handle missing records
    MissingRecordStorage: true,
}
```

### Custom Key Serialization

Implement your own key generation strategy:

```go
type CustomKeySerializer struct {
    prefix string
}

func (s *CustomKeySerializer) SerializeKey(method string, args ...any) string {
    // Your custom logic here
    return fmt.Sprintf("%s:%s:%s", s.prefix, method, /* serialize args */)
}

// Use with repository
cachedRepo := repositorycache.New(baseRepo, cacheService, &CustomKeySerializer{
    prefix: "myapp:v1",
})
```

## Performance Benefits

With in memory caching:
- **Cache hits**: ~100-500 nanoseconds
- **Database queries**: ~1-50 milliseconds
- **Improvement**: 10,000x - 100,000x faster for repeated reads

Additional benefits:
- Reduced database load
- Improved application response times
- Built in stampede protection prevents cache dogpiling

## Cache Invalidation

Write operations automatically invalidate cached reads when the cache service
implements `cache.TagRegistry` (the default sturdyc adapter does). The decorator
registers read keys under scope, ID, identifier, and list tags, then invalidates
those tags after creates/updates/deletes.

If `TagRegistry` is not available, the decorator falls back to prefix deletion for
List/Count/Get caches.

Need custom grouping? Attach extra tags to any read path:

```go
ctx := repositorycache.WithCacheTags(ctx, "preferences:tenant:"+tenantID)
prefs, total, err := cachedRepo.List(ctx, repository.Where("tenant_id", tenantID))
```

## Examples

### Complete Example

See [`cmd/app/main.go`](cmd/app/main.go) for a full working example that demonstrates:
- Setting up the cache infrastructure
- Wrapping a fake repository
- Showing cache hits vs misses
- Performance measurement

Run the example:

```bash
go run cmd/app/main.go
```

### Testing Your Cached Repository

```go
func TestCachedRepository(t *testing.T) {
    // Use an in memory cache for testing
    config := cache.Config{TTL: 5 * time.Minute}
    container, err := di.NewContainer(config)
    if err != nil {
        t.Fatalf("failed to create container: %v", err)
    }

    baseRepo := &mockUserRepository{}
    cachedRepo := container.NewCachedRepository(baseRepo)

    // test cache behavior
    user1, _ := cachedRepo.GetByID(ctx, "123") // Calls base repo
    user2, _ := cachedRepo.GetByID(ctx, "123") // Uses cache

    assert.Equal(t, user1, user2)
    assert.Equal(t, 1, baseRepo.CallCount) // Only one DB call
}
```
## License

This project is licensed under the MIT License, see the [LICENSE](LICENSE) file for details.

## Related Projects

- [go-repository-bun](https://github.com/goliatone/go-repository-bun): The base repository interface
- [sturdyc](https://github.com/viccon/sturdyc): The underlying cache library
- [uptrace/bun](https://github.com/uptrace/bun): The SQL client
