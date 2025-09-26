# Simple Repository Cache Example

This example demonstrates the basic usage of the `go-repository-cache` library with a fake repository implementation.

## What this example shows

- Setting up a DI container with cache configuration
- Creating a cached repository wrapper around a base repository
- Cache miss vs hit performance comparison
- Cache invalidation after write operations
- Data consistency through proper invalidation

## Running the example

```bash
go run main.go
```

## Key concepts demonstrated

- **Cache Performance**: Shows ~100x performance improvement on cache hits
- **Cache Invalidation**: Write operations properly invalidate affected cache entries
- **Data Consistency**: Fresh data is fetched after cache invalidation
- **Interface Compatibility**: Works seamlessly with `go-repository-bun` interfaces

## Expected output

The example will show:
1. DI container setup with cache configuration
2. Cache MISS followed by cache HIT for the same operation
3. Performance comparison between database calls and cache hits
4. Cache invalidation after create/update operations
5. Verification that fresh data is fetched after invalidation

The fake repository simulates database latency (75-150ms) to clearly demonstrate the caching benefits.