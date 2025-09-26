package cacheinfra

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/viccon/sturdyc"
)

// Config holds the configuration for the sturdyc cache adapter.
// It encapsulates the core sturdyc options needed for cache initialization.
type Config struct {
	// Capacity defines the maximum number of entries that the cache can store.
	// Must be greater than 0.
	Capacity int

	// NumShards determines the number of cache shards for concurrent access.
	// Higher values improve concurrency but increase memory overhead.
	// Must be greater than 0. Default: 256
	NumShards int

	// TTL is the default time-to-live for cached entries.
	// After this duration, entries are considered expired.
	// Must be greater than 0.
	TTL time.Duration

	// EvictionPercentage specifies what percentage of entries to evict
	// when the cache reaches its capacity. Must be between 1-100.
	// Default: 10 (evict 10% of entries)
	EvictionPercentage int

	// EarlyRefresh configures early refresh behavior for cached entries.
	// If nil, early refresh is disabled.
	EarlyRefresh *EarlyRefreshConfig

	// MissingRecordStorage enables storage for missing record flags.
	// When enabled, the cache will remember keys that returned no results
	// to prevent repeated database queries for non-existent records.
	MissingRecordStorage bool

	// EvictionInterval sets how often the cache checks for expired entries.
	// Zero value uses the default interval.
	EvictionInterval time.Duration
}

// EarlyRefreshConfig configures early refresh behavior.
// Early refresh prevents cache stampedes by refreshing entries
// before they expire when they're frequently accessed.
type EarlyRefreshConfig struct {
	// MinAsyncRefreshTime is the minimum time after which an async refresh can occur
	MinAsyncRefreshTime time.Duration

	// MaxAsyncRefreshTime is the maximum time after which an async refresh can occur
	MaxAsyncRefreshTime time.Duration

	// SyncRefreshTime is when a refresh becomes synchronous instead of async
	SyncRefreshTime time.Duration

	// RetryBaseDelay is the base delay for retry attempts when early refresh fails
	RetryBaseDelay time.Duration
}

// DefaultConfig returns a Config with sensible defaults for most use cases.
func DefaultConfig() Config {
	return Config{
		Capacity:           10000,
		NumShards:          256,
		TTL:                5 * time.Minute,
		EvictionPercentage: 10,
		EarlyRefresh: &EarlyRefreshConfig{
			MinAsyncRefreshTime: 10 * time.Second,
			MaxAsyncRefreshTime: 20 * time.Second,
			SyncRefreshTime:     30 * time.Second,
			RetryBaseDelay:      100 * time.Millisecond,
		},
		MissingRecordStorage: true,
		EvictionInterval:     0, // Use default
	}
}

// ToSturdycOptions converts the Config to sturdyc.Option slice.
// This method maps our configuration parameters to the sturdyc options.
// Note: Capacity, NumShards, TTL, and EvictionPercentage are passed directly
// to sturdyc.New() constructor and are not included in the options.
func (c Config) ToSturdycOptions() []sturdyc.Option {
	var options []sturdyc.Option

	// Configure early refresh if specified
	if c.EarlyRefresh != nil {
		options = append(options, sturdyc.WithEarlyRefreshes(
			c.EarlyRefresh.MinAsyncRefreshTime,
			c.EarlyRefresh.MaxAsyncRefreshTime,
			c.EarlyRefresh.SyncRefreshTime,
			c.EarlyRefresh.RetryBaseDelay,
		))
	}

	// Configure missing record storage
	if c.MissingRecordStorage {
		options = append(options, sturdyc.WithMissingRecordStorage())
	}

	// Configure eviction interval if specified
	if c.EvictionInterval > 0 {
		options = append(options, sturdyc.WithEvictionInterval(c.EvictionInterval))
	}

	return options
}

// Validate checks if the configuration values are valid.
// Returns an error if any configuration parameter is invalid.
func (c Config) Validate() error {
	if c.Capacity <= 0 {
		return &ConfigError{Field: "Capacity", Message: "must be greater than 0"}
	}

	if c.NumShards <= 0 {
		return &ConfigError{Field: "NumShards", Message: "must be greater than 0"}
	}

	if c.TTL <= 0 {
		return &ConfigError{Field: "TTL", Message: "must be greater than 0"}
	}

	if c.EvictionPercentage < 1 || c.EvictionPercentage > 100 {
		return &ConfigError{Field: "EvictionPercentage", Message: "must be between 1 and 100"}
	}

	if c.EarlyRefresh != nil {
		if c.EarlyRefresh.MinAsyncRefreshTime < 0 {
			return &ConfigError{Field: "EarlyRefresh.MinAsyncRefreshTime", Message: "must be non-negative"}
		}
		if c.EarlyRefresh.MaxAsyncRefreshTime < 0 {
			return &ConfigError{Field: "EarlyRefresh.MaxAsyncRefreshTime", Message: "must be non-negative"}
		}
		if c.EarlyRefresh.SyncRefreshTime < 0 {
			return &ConfigError{Field: "EarlyRefresh.SyncRefreshTime", Message: "must be non-negative"}
		}
		if c.EarlyRefresh.RetryBaseDelay < 0 {
			return &ConfigError{Field: "EarlyRefresh.RetryBaseDelay", Message: "must be non-negative"}
		}
	}

	return nil
}

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	return "config error in field " + e.Field + ": " + e.Message
}

// sturdycService wraps a sturdyc client providing caching behaviour.
type sturdycService struct {
	client *sturdyc.Client[any]
}

// NewSturdycService creates a new sturdyc cache service adapter.
// It validates the configuration and initializes a sturdyc client with the provided settings.
//
// The constructor translates Config parameters to sturdyc initialization:
// - Capacity, NumShards, TTL, EvictionPercentage are passed to sturdyc.New()
// - Other options are applied via ToSturdycOptions()
//
// Version compatibility note: This implementation assumes sturdyc v1.x API.
// Monitor sturdyc version upgrades for potential option mapping changes.
func NewSturdycService(cfg Config) (*sturdycService, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create sturdyc client with core parameters
	client := sturdyc.New[any](
		cfg.Capacity,
		cfg.NumShards,
		cfg.TTL,
		cfg.EvictionPercentage,
		cfg.ToSturdycOptions()...,
	)

	return &sturdycService{client: client}, nil
}

// GetOrFetch implements cache.CacheService.GetOrFetch.
// It attempts to retrieve a value from the cache using the provided key.
// If the key is not found or expired, it executes the fetchFn to get a fresh value,
// stores it in the cache, and returns it.
//
// The fetchFn parameter must be of type cache.FetchFn[T] where T matches the expected return type.
// Generic type inference is handled by the sturdyc client.
// validateFetchFn performs comprehensive validation of the fetchFn parameter
// to ensure it matches the expected signature: func(context.Context) (T, error)
func validateFetchFn(fetchFn any) error {
	if fetchFn == nil {
		return &ConfigError{Field: "fetchFn", Message: "cannot be nil"}
	}

	fnValue := reflect.ValueOf(fetchFn)
	fnType := fnValue.Type()

	// Validate function signature: func(context.Context) (T, error)
	if fnType.Kind() != reflect.Func {
		return &ConfigError{Field: "fetchFn", Message: "must be a function"}
	}

	if fnType.NumIn() != 1 || fnType.NumOut() != 2 {
		return &ConfigError{Field: "fetchFn", Message: "must have signature func(context.Context) (T, error)"}
	}

	// Check input parameter is context.Context
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !fnType.In(0).Implements(contextType) {
		return &ConfigError{Field: "fetchFn", Message: "first parameter must be context.Context"}
	}

	// Check second output parameter is error
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(1).Implements(errorType) {
		return &ConfigError{Field: "fetchFn", Message: "second return value must be error"}
	}

	return nil
}

func (s *sturdycService) GetOrFetch(ctx context.Context, key string, fetchFn any) (any, error) {
	// Validate fetchFn completely before calling sturdyc to avoid ErrInvalidType conversion
	if err := validateFetchFn(fetchFn); err != nil {
		return nil, err
	}

	// Use reflection to create a wrapper that calls the generic fetchFn
	// and returns the result as any type for sturdyc compatibility
	typedFetchFn := func(ctx context.Context) (any, error) {
		return callFetchFunctionWithReflection(ctx, fetchFn)
	}

	// Use sturdyc's GetOrFetch with the typed function
	return s.client.GetOrFetch(ctx, key, typedFetchFn)
}

// callFetchFunctionWithReflection uses reflection to call any function that matches
// the FetchFn[T] signature: func(context.Context) (T, error)
// Note: fetchFn is guaranteed to be valid as it's pre validated by validateFetchFn
func callFetchFunctionWithReflection(ctx context.Context, fetchFn any) (any, error) {
	// First try direct type assertion for the common case
	if fn, ok := fetchFn.(func(context.Context) (any, error)); ok {
		return fn(ctx)
	}

	// Use reflection to handle generic functions (validation already done)
	fnValue := reflect.ValueOf(fetchFn)

	// Call the function using reflection
	results := fnValue.Call([]reflect.Value{reflect.ValueOf(ctx)})

	// Extract results
	var result any
	var err error

	// Get the first return value (T)
	resultValue := results[0]
	if resultValue.IsValid() {
		if resultValue.CanInterface() {
			result = resultValue.Interface()
		}
	}

	// Get the second return value (error)
	errorValue := results[1]
	if errorValue.IsValid() && !errorValue.IsNil() {
		err = errorValue.Interface().(error)
	}

	return result, err
}

// Delete implements cache.CacheService.Delete.
// Removes a single entry from the cache using the provided key.
// This ensures subsequent GetOrFetch calls will fetch fresh data from the source.
func (s *sturdycService) Delete(ctx context.Context, key string) error {
	s.client.Delete(key)
	return nil
}

// DeleteByPrefix implements cache.CacheService.DeleteByPrefix.
// Removes all entries from the cache that have keys starting with the given prefix.
// This is useful for invalidating related cache entries (e.g., all entries for a specific entity).
func (s *sturdycService) DeleteByPrefix(ctx context.Context, prefix string) error {
	// Get all keys from the cache
	keys := s.client.ScanKeys()

	// Delete keys that match the prefix
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			s.client.Delete(key)
		}
	}

	return nil
}

// InvalidateKeys implements cache.CacheService.InvalidateKeys.
// Removes multiple entries from the cache using the provided keys.
// This method provides an efficient way to invalidate multiple related cache entries
// in a single operation.
func (s *sturdycService) InvalidateKeys(ctx context.Context, keys []string) error {
	for _, key := range keys {
		s.client.Delete(key)
	}
	return nil
}
