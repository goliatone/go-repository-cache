# go-repository-cache

A **type-safe caching decorator** for [go-repository-bun](https://github.com/goliatone/go-repository-bun) repositories using [sturdyc](https://github.com/viccon/sturdyc) for stampede-safe caching.

## Features

- **Drop-in compatibility** - Implements the same `Repository[T]` interface
- **Stampede protection** - Non-blocking reads with in-flight request deduplication
- **Type-safe caching** - Generic implementation maintains full type safety
- **Selective caching** - Only read operations are cached; writes pass through
- **Transaction awareness** - Bypasses cache for transactional operations
- **Smart key generation** - Handles complex criteria including function pointers
- **Configurable** - Pluggable key serialization and cache configuration

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
// These generate different cache keys:
repo.GetByID(ctx, "123")
repo.GetByID(ctx, "123", repository.WithDeleted())
repo.List(ctx, repository.Where("active", true))
repo.List(ctx, repository.Where("active", false))
```

### Transaction Handling

Operations within transactions bypass the cache to ensure consistency:

```go
// This hits the cache
user, err := repo.GetByID(ctx, "123")

// This bypasses the cache (goes directly to DB)
err = repo.WithTx(ctx, func(ctx context.Context, tx bun.IDB) error {
    user, err := repo.GetByIDTx(ctx, tx, "123") // Direct DB access
    return repo.UpdateTx(ctx, tx, updatedUser)
})
```

## Configuration

### Cache Configuration

```go
config := cache.Config{
    // Basic settings
    TTL:                30 * time.Minute,
    NumShards:          256,
    EvictionPercentage: 10,

    // Early refresh (background refresh before expiry)
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

Currently, write operations don't automatically invalidate cache entries. This is intentional to keep the implementation simple and predictable. Future versions will support:

- Pattern-based invalidation (`User:*`)
- Tag-based cache grouping
- Time-based expiration strategies
- Manual invalidation APIs

For now, you can work around this by:
1. Using shorter TTLs for frequently updated data
2. Implementing custom invalidation in your service layer
3. Using cache tags if your backend supports them

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

    baseRepo := &mockUserRepository{} // Your test double
    cachedRepo := container.NewCachedRepository(baseRepo)

    // Test cache behavior
    user1, _ := cachedRepo.GetByID(ctx, "123") // Calls base repo
    user2, _ := cachedRepo.GetByID(ctx, "123") // Uses cache

    assert.Equal(t, user1, user2)
    assert.Equal(t, 1, baseRepo.CallCount) // Only one DB call
}
```
## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [go-repository-bun](https://github.com/goliatone/go-repository-bun) - The base repository interface
- [sturdyc](https://github.com/viccon/sturdyc) - The underlying cache library
- [uptrace/bun](https://github.com/uptrace/bun) - The SQL client
