package di

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/goliatone/go-repository-cache/cache"
	"github.com/goliatone/go-repository-cache/internal/cacheinfra"
)

// TestConcurrentAccess tests concurrent access to cached repository operations
func TestConcurrentAccess(t *testing.T) {
	config := cacheinfra.Config{
		Capacity:             1000,
		NumShards:            16,
		TTL:                  5 * time.Second,
		EvictionPercentage:   10,
		EarlyRefresh:         nil,
		MissingRecordStorage: true,
		EvictionInterval:     0,
	}

	container, err := NewContainer(config)
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	mockRepo := newMockUserRepository()
	cachedRepo := NewCachedRepository(container, mockRepo)

	// Pre-populate with test data
	testUsers := make([]User, 100)
	for i := 0; i < 100; i++ {
		user := User{
			ID:       fmt.Sprintf("user-%d", i),
			Name:     fmt.Sprintf("User %d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			CreateTs: time.Now().Unix(),
		}
		testUsers[i] = user
		mockRepo.Create(context.Background(), user)
	}

	ctx := context.Background()
	const numGoroutines = 50
	const operationsPerGoroutine = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Launch concurrent workers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				userID := fmt.Sprintf("user-%d", (workerID*operationsPerGoroutine+j)%100)

				// Perform GetByID operation
				_, err := cachedRepo.GetByID(ctx, userID)
				if err != nil {
					errors <- fmt.Errorf("worker %d operation %d GetByID failed: %v", workerID, j, err)
					continue
				}

				// Perform List operation every 5th iteration
				if j%5 == 0 {
					_, _, err := cachedRepo.List(ctx)
					if err != nil {
						errors <- fmt.Errorf("worker %d operation %d List failed: %v", workerID, j, err)
						continue
					}
				}

				// Perform Count operation every 10th iteration
				if j%10 == 0 {
					_, err := cachedRepo.Count(ctx)
					if err != nil {
						errors <- fmt.Errorf("worker %d operation %d Count failed: %v", workerID, j, err)
						continue
					}
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	var errorCount int
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 10 { // Limit error output
			t.Error("... and more errors")
			break
		}
	}

	if errorCount > 0 {
		t.Fatalf("Concurrent access test failed with %d errors", errorCount)
	}

	// Verify that caching is working (base repository should be called much less than total operations)
	totalOperations := numGoroutines * operationsPerGoroutine
	getByIDCalls := mockRepo.getCallCount("GetByID")

	if getByIDCalls >= totalOperations {
		t.Errorf("Expected cache to reduce GetByID calls: got %d calls for %d operations", getByIDCalls, totalOperations)
	}

	t.Logf("Concurrent test completed: %d operations resulted in %d GetByID calls (%.1f%% cache hit rate)",
		totalOperations, getByIDCalls, float64(totalOperations-getByIDCalls)/float64(totalOperations)*100)
}

// TestConcurrentReadWrite tests concurrent read and write operations
func TestConcurrentReadWrite(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	mockRepo := newMockUserRepository()
	cachedRepo := NewCachedRepository(container, mockRepo)

	ctx := context.Background()
	const numReaders = 10
	const numWriters = 5
	const operationsPerWorker = 20

	var wg sync.WaitGroup
	errors := make(chan error, (numReaders+numWriters)*operationsPerWorker)

	// Launch reader workers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				userID := fmt.Sprintf("read-user-%d", readerID)

				_, err := cachedRepo.GetByID(ctx, userID)
				// It's okay if user doesn't exist, we're testing concurrency
				if err != nil && err.Error() != "user not found" {
					errors <- fmt.Errorf("reader %d operation %d failed: %v", readerID, j, err)
				}

				time.Sleep(1 * time.Millisecond) // Small delay to increase contention
			}
		}(i)
	}

	// Launch writer workers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				user := User{
					ID:       fmt.Sprintf("write-user-%d-%d", writerID, j),
					Name:     fmt.Sprintf("Writer %d User %d", writerID, j),
					Email:    fmt.Sprintf("writer%d.%d@example.com", writerID, j),
					CreateTs: time.Now().Unix(),
				}

				_, err := cachedRepo.Create(ctx, user)
				if err != nil {
					errors <- fmt.Errorf("writer %d operation %d failed: %v", writerID, j, err)
				}

				time.Sleep(2 * time.Millisecond) // Small delay
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 5 {
			t.Error("... and more errors")
			break
		}
	}

	if errorCount > 0 {
		t.Errorf("Concurrent read-write test had %d errors", errorCount)
	}
}

// TestTTLExpiryIntegration tests cache entries expiring based on TTL settings
func TestTTLExpiryIntegration(t *testing.T) {
	shortTTLConfig := cacheinfra.Config{
		Capacity:             50,
		NumShards:            4,
		TTL:                  200 * time.Millisecond,
		EvictionPercentage:   10,
		EarlyRefresh:         nil,
		MissingRecordStorage: true,
		EvictionInterval:     50 * time.Millisecond,
	}

	container, err := NewContainer(shortTTLConfig)
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	mockRepo := newMockUserRepository()
	cachedRepo := NewCachedRepository(container, mockRepo)

	// Create test data
	testUser := User{
		ID:       "ttl-test-user",
		Name:     "TTL Test User",
		Email:    "ttl@example.com",
		CreateTs: time.Now().Unix(),
	}
	mockRepo.Create(context.Background(), testUser)

	ctx := context.Background()

	// Phase 1: Initial cache population
	_, err = cachedRepo.GetByID(ctx, "ttl-test-user")
	if err != nil {
		t.Fatalf("Initial GetByID failed: %v", err)
	}

	initialCalls := mockRepo.getCallCount("GetByID")
	if initialCalls != 1 {
		t.Errorf("Expected 1 initial GetByID call, got %d", initialCalls)
	}

	// Phase 2: Immediate re-access (should be cached)
	_, err = cachedRepo.GetByID(ctx, "ttl-test-user")
	if err != nil {
		t.Fatalf("Cached GetByID failed: %v", err)
	}

	cachedCalls := mockRepo.getCallCount("GetByID")
	if cachedCalls != 1 {
		t.Errorf("Expected cached access to not increase calls, got %d", cachedCalls)
	}

	// Phase 3: Wait for TTL expiry
	time.Sleep(300 * time.Millisecond) // Wait longer than TTL

	// Phase 4: Access after expiry (should hit base repository again)
	_, err = cachedRepo.GetByID(ctx, "ttl-test-user")
	if err != nil {
		t.Fatalf("Post-expiry GetByID failed: %v", err)
	}

	expiredCalls := mockRepo.getCallCount("GetByID")
	if expiredCalls != 2 {
		t.Errorf("Expected 2 calls after TTL expiry, got %d", expiredCalls)
	}

	t.Logf("TTL expiry test successful: %d calls total", expiredCalls)
}

// TestBatchOperationsIntegration tests scenarios with batch operations
func TestBatchOperationsIntegration(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	mockRepo := newMockUserRepository()
	cachedRepo := NewCachedRepository(container, mockRepo)

	ctx := context.Background()

	// Create batch of users
	batchSize := 50
	users := make([]User, batchSize)
	for i := 0; i < batchSize; i++ {
		user := User{
			ID:       fmt.Sprintf("batch-user-%d", i),
			Name:     fmt.Sprintf("Batch User %d", i),
			Email:    fmt.Sprintf("batch%d@example.com", i),
			CreateTs: time.Now().Unix(),
		}
		users[i] = user
		mockRepo.Create(ctx, user)
	}

	// First batch read - should populate cache
	for i := 0; i < batchSize; i++ {
		_, err := cachedRepo.GetByID(ctx, fmt.Sprintf("batch-user-%d", i))
		if err != nil {
			t.Fatalf("Batch read failed for user %d: %v", i, err)
		}
	}

	firstBatchCalls := mockRepo.getCallCount("GetByID")
	if firstBatchCalls != batchSize {
		t.Errorf("Expected %d calls for first batch, got %d", batchSize, firstBatchCalls)
	}

	// Second batch read - should be served from cache
	for i := 0; i < batchSize; i++ {
		_, err := cachedRepo.GetByID(ctx, fmt.Sprintf("batch-user-%d", i))
		if err != nil {
			t.Fatalf("Cached batch read failed for user %d: %v", i, err)
		}
	}

	secondBatchCalls := mockRepo.getCallCount("GetByID")
	if secondBatchCalls != batchSize {
		t.Errorf("Expected cached reads to not increase calls, got %d", secondBatchCalls)
	}

	t.Logf("Batch operations test completed: %d users, %d repository calls", batchSize, secondBatchCalls)
}

// BenchmarkKeySerializationPerformance benchmarks key serialization performance
func BenchmarkKeySerializationPerformance(b *testing.B) {
	serializer := cache.NewDefaultKeySerializer()

	testCases := []struct {
		name string
		args []any
	}{
		{
			name: "simple_args",
			args: []any{"test-id", 123, true},
		},
		{
			name: "complex_struct",
			args: []any{
				User{
					ID:       "bench-user",
					Name:     "Benchmark User",
					Email:    "bench@example.com",
					CreateTs: time.Now().Unix(),
				},
			},
		},
		{
			name: "slice_args",
			args: []any{[]string{"a", "b", "c"}, []int{1, 2, 3, 4, 5}},
		},
		{
			name: "map_args",
			args: []any{
				map[string]any{
					"key1": "value1",
					"key2": 42,
					"key3": true,
				},
			},
		},
		{
			name: "mixed_complex",
			args: []any{
				"method",
				User{ID: "test"},
				[]string{"filter1", "filter2"},
				map[string]int{"limit": 10, "offset": 0},
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = serializer.SerializeKey("GetByID", tc.args...)
			}
		})
	}
}

// BenchmarkCachedVsBaseRepository compares performance of cached vs base repository operations
func BenchmarkCachedVsBaseRepository(b *testing.B) {
	// Setup
	container, err := NewContainerWithDefaults()
	if err != nil {
		b.Fatalf("Failed to create DI container: %v", err)
	}

	mockRepo := newMockUserRepository()
	cachedRepo := NewCachedRepository(container, mockRepo)

	// Pre-populate with test data
	testUsers := make([]User, 1000)
	for i := 0; i < 1000; i++ {
		user := User{
			ID:       fmt.Sprintf("bench-user-%d", i),
			Name:     fmt.Sprintf("Benchmark User %d", i),
			Email:    fmt.Sprintf("bench%d@example.com", i),
			CreateTs: time.Now().Unix(),
		}
		testUsers[i] = user
		mockRepo.Create(context.Background(), user)
	}

	ctx := context.Background()

	b.Run("base_repository_GetByID", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("bench-user-%d", i%1000)
			_, _ = mockRepo.GetByID(ctx, userID)
		}
	})

	b.Run("cached_repository_GetByID_first_access", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("first-access-user-%d", i)
			user := User{
				ID:       userID,
				Name:     fmt.Sprintf("First Access User %d", i),
				Email:    fmt.Sprintf("first%d@example.com", i),
				CreateTs: time.Now().Unix(),
			}
			mockRepo.Create(ctx, user)
			_, _ = cachedRepo.GetByID(ctx, userID)
		}
	})

	// Warm up cache for cached access benchmark
	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("bench-user-%d", i)
		cachedRepo.GetByID(ctx, userID)
	}

	b.Run("cached_repository_GetByID_cache_hit", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("bench-user-%d", i%100) // Use warmed up entries
			_, _ = cachedRepo.GetByID(ctx, userID)
		}
	})

	b.Run("base_repository_List", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = mockRepo.List(ctx)
		}
	})

	// Warm up cache for List
	cachedRepo.List(ctx)

	b.Run("cached_repository_List_cache_hit", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = cachedRepo.List(ctx)
		}
	})

	b.Run("base_repository_Count", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockRepo.Count(ctx)
		}
	})

	// Warm up cache for Count
	cachedRepo.Count(ctx)

	b.Run("cached_repository_Count_cache_hit", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cachedRepo.Count(ctx)
		}
	})
}

// generateComplexArgs helper function for benchmarks
func generateComplexArgs(depth int) []any {
	if depth == 0 {
		return []any{"simple", 123}
	}

	nested := make(map[string]any)
	nested["depth"] = depth
	nested["slice"] = make([]any, depth*2)
	for i := 0; i < depth*2; i++ {
		nested["slice"].([]any)[i] = fmt.Sprintf("item-%d", i)
	}

	if depth > 1 {
		nested["nested"] = generateComplexArgs(depth - 1)
	}

	return []any{nested}
}

// BenchmarkCacheKeyGenerationComplexity benchmarks key generation with varying complexity
func BenchmarkCacheKeyGenerationComplexity(b *testing.B) {
	serializer := cache.NewDefaultKeySerializer()

	complexityLevels := []int{1, 3, 5, 7, 10}
	for _, level := range complexityLevels {
		b.Run(fmt.Sprintf("complexity_level_%d", level), func(b *testing.B) {
			args := generateComplexArgs(level)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = serializer.SerializeKey("ComplexMethod", args...)
			}
		})
	}
}

// BenchmarkConcurrentCacheAccess benchmarks performance under concurrent load
func BenchmarkConcurrentCacheAccess(b *testing.B) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		b.Fatalf("Failed to create DI container: %v", err)
	}

	mockRepo := newMockUserRepository()
	cachedRepo := NewCachedRepository(container, mockRepo)

	// Pre-populate
	for i := 0; i < 100; i++ {
		user := User{
			ID:       fmt.Sprintf("concurrent-user-%d", i),
			Name:     fmt.Sprintf("Concurrent User %d", i),
			Email:    fmt.Sprintf("concurrent%d@example.com", i),
			CreateTs: time.Now().Unix(),
		}
		mockRepo.Create(context.Background(), user)
		cachedRepo.GetByID(context.Background(), user.ID) // Warm cache
	}

	ctx := context.Background()

	b.Run("concurrent_cache_hits", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				userID := fmt.Sprintf("concurrent-user-%d", i%100)
				_, _ = cachedRepo.GetByID(ctx, userID)
				i++
			}
		})
	})
}
