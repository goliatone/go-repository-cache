// Package cache provides caching interfaces and key serialization for repository caching.
//
// # Overview
//
// This package exports two main interfaces and their default implementations:
//
//   - CacheService: A generic caching interface for read-through operations
//   - KeySerializer: Builds stable cache keys from method names and arguments
//
// The cache package is designed to work with repository decorators that need to cache
// read operations while maintaining type safety through generics.
//
// # Basic Usage
//
// The simplest way to use the cache package is with the default key serializer:
//
//	serializer := cache.NewDefaultKeySerializer()
//	key := serializer.SerializeKey("GetByID", "user-123", someCriteriaFunc)
//
// For repository caching, you would typically use this with a CacheService implementation:
//
//	// Assume you have a cache service implementation
//	result, err := cache.GetOrFetch(ctx, cacheService, key, func(ctx context.Context) (User, error) {
//		return repository.GetByID(ctx, "user-123", someCriteriaFunc)
//	})
//
// # Key Serialization Strategy
//
// The default key serializer uses reflection to handle various Go types:
//
//   - Function pointers: Uses %p formatting for stability within a process
//   - Basic types: Direct string representation
//   - Slices/arrays: Recursive serialization of elements
//   - Maps: Sorted key-value pairs for deterministic output
//   - Structs: Exported fields with name:value pairs
//   - Complex types: JSON fallback with error handling
//
// # Important Warnings for Function Criteria
//
// When using function criteria (common in repository patterns), be aware of these limitations:
//
//   - Function pointers are stable only within a single process lifetime
//   - Closures with different captured variables will have different pointers
//   - Anonymous functions created at different call sites will have different pointers
//   - For distributed caching, consider a custom KeySerializer that includes stable criteria names
//
// # Custom Key Serializers
//
// You can implement your own KeySerializer for specialized key generation:
//
//	type CustomKeySerializer struct {
//		prefix string
//	}
//
//	func (s *CustomKeySerializer) SerializeKey(method string, args ...any) string {
//		// Custom logic here
//		return s.prefix + ":" + method + ":" + /* serialize args */
//	}
//
// This is useful when you need:
//   - Different key formats for different cache backends
//   - Stable keys across process restarts for function criteria
//   - Application-specific key structures or namespacing
//
// # Error Handling
//
// The package prioritizes stability over perfection. When JSON marshaling fails,
// the key serializer falls back to type information and memory addresses rather
// than panicking. This ensures cache operations continue even with problematic data types.
//
// # See Also
//
// For complete usage examples with repository decorators, see the repositorycache package.
// For the specific key generation implementation details, see key_serializer.go.
package cache
