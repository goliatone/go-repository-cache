// Package repositorycache provides cached repository decorators for go-repository-bun.
//
// # Overview
//
// This package implements the repository decorator pattern to add caching capabilities
// to existing repository implementations from go-repository-bun. The cached repository
// wraps a base repository and intercepts read operations to provide caching while
// delegating write operations directly to the base repository.
//
// # Key Features
//
//   - **Type-safe caching**: Uses Go generics to maintain type safety across cached operations
//   - **Selective caching**: Only read operations are cached; write operations pass through
//   - **Transaction awareness**: Transaction-based operations bypass cache for consistency
//   - **Pluggable key strategy**: Configurable key serialization via KeySerializer interface
//   - **Cache invalidation ready**: Structured for future cache invalidation strategies
//
// # Basic Usage
//
// Create a cached repository by wrapping an existing repository:
//
//	base := myrepo.New(db) // Your existing go-repository-bun repository
//	cache := internal.NewSturdycService(cacheConfig)
//	keySerializer := cache.NewDefaultKeySerializer()
//
//	cached := repositorycache.New(base, cache, keySerializer)
//
//	// Use exactly like your base repository
//	user, err := cached.GetByID(ctx, "user-123")
//	users, total, err := cached.List(ctx, repository.Where("active", true))
//
// # Cached vs Pass-through Operations
//
// ## Cached Operations (Read-only)
//
// These operations use the cache for improved performance:
//   - Get, GetByID, GetByIdentifier
//   - List, Count
//
// ## Pass-through Operations
//
// These operations bypass the cache and go directly to the base repository:
//   - All write operations (Create, Update, Upsert, Delete and variants)
//   - All transaction-based operations (*Tx methods)
//   - Raw SQL queries
//
// # Caching Behavior
//
// The cached repository follows a read-through caching pattern:
//
//  1. Check cache for the serialized key
//  2. If cache hit, return cached result
//  3. If cache miss, call base repository
//  4. Store result in cache
//  5. Return result to caller
//
// Key serialization includes the method name and all parameters to ensure cache
// correctness across different query patterns.
//
// # Transaction Handling
//
// Operations within transactions (*Tx methods) bypass the cache entirely to ensure
// transaction isolation and consistency. This prevents:
//   - Reading stale cached data within transactions
//   - Cache pollution from uncommitted transaction data
//   - Inconsistent reads across transaction boundaries
//
// # Cache Invalidation Strategy
//
// A comprehensive cache invalidation strategy has been designed and documented
// in REPOSITORY_CACHE.md. The strategy addresses:
//
//   - **Key structure analysis**: Understanding cache key patterns for targeted invalidation
//   - **Operation-specific invalidation**: Different strategies for Create, Update, Delete operations
//   - **Key registry pattern**: Tracking active cache keys for prefix-based invalidation
//   - **Reflection-based helpers**: Extracting IDs and identifiers from records for targeted cache clearing
//   - **Testing fixtures**: Comprehensive scenarios for invalidation behavior validation
//
// Implementation is planned across tasks TSK-20 through TSK-22:
//   - TSK-20: Extend CacheService with prefix-based invalidation methods
//   - TSK-21: Implement key registry and invalidation logic in the decorator
//   - TSK-22: Add comprehensive testing and edge case handling
//
// # Integration with Dependency Injection
//
// This package is designed to work with the dependency injection container
// provided in pkg/di:
//
//	container, err := di.NewContainer(cacheConfig)
//	if err != nil {
//		return err
//	}
//	cachedRepo := container.NewCachedRepository(baseRepo)
//
// # Compatibility
//
// The CachedRepository[T] fully implements the repository.Repository[T] interface
// from go-repository-bun, making it a drop-in replacement for existing repository
// usage. The decorator pattern ensures that all methods are available and maintain
// the same signatures as the base interface.
//
// # Performance Considerations
//
//   - Cache hits avoid database roundtrips for read operations
//   - Key serialization has minimal overhead using reflection
//   - List operations cache both records and total count as a unit
//   - Function criteria in keys use pointer addresses (stable per process)
//
// # Error Handling
//
// Errors from the base repository are propagated unchanged. Cache errors
// (serialization failures, cache backend issues) are handled gracefully
// without breaking the underlying repository operations.
//
// # See Also
//
// For cache configuration and key serialization details, see the cache package.
// For dependency injection setup, see the pkg/di package.
package repositorycache
