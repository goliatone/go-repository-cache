package di

import (
	"context"
	"testing"
	"time"

	"github.com/goliatone/go-repository-cache/cache"
)

func TestNewContainer(t *testing.T) {
	config := cache.Config{
		Capacity:           1000,
		NumShards:          256,
		TTL:                5 * time.Minute,
		EvictionPercentage: 10,
		EarlyRefresh: &cache.EarlyRefreshConfig{
			MinAsyncRefreshTime: 10 * time.Second,
			MaxAsyncRefreshTime: 20 * time.Second,
			SyncRefreshTime:     30 * time.Second,
			RetryBaseDelay:      100 * time.Millisecond,
		},
		MissingRecordStorage: true,
		EvictionInterval:     0,
	}

	container, err := NewContainer(config)
	if err != nil {
		t.Fatalf("NewContainer() failed: %v", err)
	}

	if container == nil {
		t.Fatal("NewContainer() returned nil container")
	}

	// Verify that dependencies are properly initialized
	if container.CacheService() == nil {
		t.Error("Container should have a non-nil cache service")
	}

	if container.KeySerializer() == nil {
		t.Error("Container should have a non-nil key serializer")
	}

	// Verify config is stored correctly
	storedConfig := container.Config()
	if storedConfig.Capacity != config.Capacity {
		t.Errorf("Expected capacity %d, got %d", config.Capacity, storedConfig.Capacity)
	}

	if storedConfig.TTL != config.TTL {
		t.Errorf("Expected TTL %v, got %v", config.TTL, storedConfig.TTL)
	}
}

func TestNewContainerWithDefaults(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("NewContainerWithDefaults() failed: %v", err)
	}

	if container == nil {
		t.Fatal("NewContainerWithDefaults() returned nil container")
	}

	// Verify that default configuration is used
	config := container.Config()
	defaultConfig := cache.DefaultConfig()

	if config.Capacity != defaultConfig.Capacity {
		t.Errorf("Expected default capacity %d, got %d", defaultConfig.Capacity, config.Capacity)
	}

	if config.TTL != defaultConfig.TTL {
		t.Errorf("Expected default TTL %v, got %v", defaultConfig.TTL, config.TTL)
	}
}

func TestNewContainer_InvalidConfig(t *testing.T) {
	invalidConfig := cache.Config{
		Capacity:           0, // Invalid: must be > 0
		NumShards:          256,
		TTL:                5 * time.Minute,
		EvictionPercentage: 10,
	}

	_, err := NewContainer(invalidConfig)
	if err == nil {
		t.Error("NewContainer() should fail with invalid config")
	}
}

func TestContainerSingletonBehavior(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("NewContainerWithDefaults() failed: %v", err)
	}

	// Call getters multiple times to ensure they return the same instances
	cacheService1 := container.CacheService()
	cacheService2 := container.CacheService()

	if cacheService1 != cacheService2 {
		t.Error("CacheService() should return the same instance (singleton behavior)")
	}

	keySerializer1 := container.KeySerializer()
	keySerializer2 := container.KeySerializer()

	if keySerializer1 != keySerializer2 {
		t.Error("KeySerializer() should return the same instance (singleton behavior)")
	}
}

func TestKeySerializerIntegration(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("NewContainerWithDefaults() failed: %v", err)
	}

	keySerializer := container.KeySerializer()

	// Test key serialization with various argument types
	testCases := []struct {
		name     string
		method   string
		args     []any
		expected string
	}{
		{
			name:     "no args",
			method:   "Get",
			args:     []any{},
			expected: "Get",
		},
		{
			name:     "single string arg",
			method:   "GetByID",
			args:     []any{"123"},
			expected: "GetByID:123",
		},
		{
			name:     "multiple args",
			method:   "List",
			args:     []any{"user", 10, true},
			expected: "List:user:10:true",
		},
		{
			name:     "nil arg",
			method:   "Count",
			args:     []any{nil},
			expected: "Count:nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := keySerializer.SerializeKey(tc.method, tc.args...)
			if result != tc.expected {
				t.Errorf("Expected key %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestCacheServiceIntegration(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("NewContainerWithDefaults() failed: %v", err)
	}

	cacheService := container.CacheService()
	ctx := context.Background()

	// Test basic cache operations
	key := "test-key"
	expectedValue := "test-value"

	// Define a fetch function that returns our test value
	fetchFn := func(ctx context.Context) (any, error) {
		return expectedValue, nil
	}

	// Get or fetch should call our function and return the value
	result, err := cacheService.GetOrFetch(ctx, key, fetchFn)
	if err != nil {
		t.Fatalf("GetOrFetch() failed: %v", err)
	}

	if result != expectedValue {
		t.Errorf("Expected value %q, got %q", expectedValue, result)
	}

	// Delete should not return an error (even if it's a no-op)
	err = cacheService.Delete(ctx, key)
	if err != nil {
		t.Errorf("Delete() failed: %v", err)
	}
}
