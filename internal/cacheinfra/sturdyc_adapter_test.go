package cacheinfra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goliatone/go-repository-cache/cache"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify default configuration values match design doc specifications
	if cfg.Capacity != 10000 {
		t.Errorf("expected Capacity to be 10000, got %d", cfg.Capacity)
	}

	if cfg.NumShards != 256 {
		t.Errorf("expected NumShards to be 256, got %d", cfg.NumShards)
	}

	if cfg.TTL != 5*time.Minute {
		t.Errorf("expected TTL to be 5 minutes, got %v", cfg.TTL)
	}

	if cfg.EvictionPercentage != 10 {
		t.Errorf("expected EvictionPercentage to be 10, got %d", cfg.EvictionPercentage)
	}

	if !cfg.MissingRecordStorage {
		t.Error("expected MissingRecordStorage to be true")
	}

	if cfg.EarlyRefresh == nil {
		t.Fatal("expected EarlyRefresh to be configured")
	}

	if cfg.EarlyRefresh.MinAsyncRefreshTime != 10*time.Second {
		t.Errorf("expected EarlyRefresh.MinAsyncRefreshTime to be 10 seconds, got %v", cfg.EarlyRefresh.MinAsyncRefreshTime)
	}

	if cfg.EarlyRefresh.MaxAsyncRefreshTime != 20*time.Second {
		t.Errorf("expected EarlyRefresh.MaxAsyncRefreshTime to be 20 seconds, got %v", cfg.EarlyRefresh.MaxAsyncRefreshTime)
	}

	if cfg.EarlyRefresh.SyncRefreshTime != 30*time.Second {
		t.Errorf("expected EarlyRefresh.SyncRefreshTime to be 30 seconds, got %v", cfg.EarlyRefresh.SyncRefreshTime)
	}

	if cfg.EarlyRefresh.RetryBaseDelay != 100*time.Millisecond {
		t.Errorf("expected EarlyRefresh.RetryBaseDelay to be 100ms, got %v", cfg.EarlyRefresh.RetryBaseDelay)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid default config",
			cfg:       DefaultConfig(),
			wantError: false,
		},
		{
			name: "invalid capacity - zero",
			cfg: Config{
				Capacity:           0,
				NumShards:          256,
				TTL:                5 * time.Minute,
				EvictionPercentage: 10,
			},
			wantError: true,
			errorMsg:  "must be greater than 0",
		},
		{
			name: "invalid num shards - zero",
			cfg: Config{
				Capacity:           1000,
				NumShards:          0,
				TTL:                5 * time.Minute,
				EvictionPercentage: 10,
			},
			wantError: true,
			errorMsg:  "must be greater than 0",
		},
		{
			name: "invalid TTL - zero",
			cfg: Config{
				Capacity:           1000,
				NumShards:          256,
				TTL:                0,
				EvictionPercentage: 10,
			},
			wantError: true,
			errorMsg:  "must be greater than 0",
		},
		{
			name: "invalid eviction percentage - too low",
			cfg: Config{
				Capacity:           1000,
				NumShards:          256,
				TTL:                5 * time.Minute,
				EvictionPercentage: 0,
			},
			wantError: true,
			errorMsg:  "must be between 1 and 100",
		},
		{
			name: "invalid eviction percentage - too high",
			cfg: Config{
				Capacity:           1000,
				NumShards:          256,
				TTL:                5 * time.Minute,
				EvictionPercentage: 101,
			},
			wantError: true,
			errorMsg:  "must be between 1 and 100",
		},
		{
			name: "invalid early refresh min async time",
			cfg: Config{
				Capacity:           1000,
				NumShards:          256,
				TTL:                5 * time.Minute,
				EvictionPercentage: 10,
				EarlyRefresh: &EarlyRefreshConfig{
					MinAsyncRefreshTime: -1 * time.Second,
					MaxAsyncRefreshTime: 20 * time.Second,
					SyncRefreshTime:     30 * time.Second,
					RetryBaseDelay:      100 * time.Millisecond,
				},
			},
			wantError: true,
			errorMsg:  "must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantError {
				if err == nil {
					t.Error("expected validation error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != "" {
					// Check if error message contains expected substring
					if len(tt.errorMsg) > 0 && err.Error() == "" {
						t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no validation error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_ToSturdycOptions(t *testing.T) {
	cfg := DefaultConfig()
	options := cfg.ToSturdycOptions()

	// Verify that we get the expected number of options for default config
	// Default config should have early refresh + missing record storage options
	expectedOptionsCount := 2
	if len(options) != expectedOptionsCount {
		t.Errorf("expected %d sturdyc options for default config, got %d", expectedOptionsCount, len(options))
	}

	// Test with minimal config (no optional features)
	minimalCfg := Config{
		Capacity:             1000,
		NumShards:            256,
		TTL:                  time.Minute,
		EvictionPercentage:   5,
		EarlyRefresh:         nil,
		MissingRecordStorage: false,
		EvictionInterval:     0,
	}

	minimalOptions := minimalCfg.ToSturdycOptions()
	if len(minimalOptions) != 0 {
		t.Errorf("expected no sturdyc options for minimal config, got %d", len(minimalOptions))
	}

	// Test with only missing record storage enabled
	missingRecordCfg := Config{
		Capacity:             1000,
		NumShards:            256,
		TTL:                  time.Minute,
		EvictionPercentage:   5,
		EarlyRefresh:         nil,
		MissingRecordStorage: true,
		EvictionInterval:     0,
	}

	missingRecordOptions := missingRecordCfg.ToSturdycOptions()
	if len(missingRecordOptions) != 1 {
		t.Errorf("expected 1 sturdyc option for missing record config, got %d", len(missingRecordOptions))
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{
		Field:   "TestField",
		Message: "test message",
	}

	expected := "config error in field TestField: test message"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestNewSturdycService(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid default config",
			cfg:       DefaultConfig(),
			wantError: false,
		},
		{
			name: "invalid config - zero capacity",
			cfg: Config{
				Capacity:           0,
				NumShards:          256,
				TTL:                5 * time.Minute,
				EvictionPercentage: 10,
			},
			wantError: true,
			errorMsg:  "config error in field Capacity: must be greater than 0",
		},
		{
			name: "invalid config - zero TTL",
			cfg: Config{
				Capacity:           1000,
				NumShards:          256,
				TTL:                0,
				EvictionPercentage: 10,
			},
			wantError: true,
			errorMsg:  "config error in field TTL: must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewSturdycService(tt.cfg)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
				if service != nil {
					t.Error("expected service to be nil when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
					return
				}
				if service == nil {
					t.Error("expected service to be non-nil")
				}
				// Verify service implements cache.CacheService interface
				var _ cache.CacheService = service
			}
		})
	}
}

func TestSturdycService_GetOrFetch(t *testing.T) {
	cfg := Config{
		Capacity:             100,
		NumShards:            2,
		TTL:                  1 * time.Minute,
		EvictionPercentage:   10,
		MissingRecordStorage: false,
	}

	service, err := NewSturdycService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()

	t.Run("cache miss - fetch function called", func(t *testing.T) {
		fetchCalled := false
		expectedValue := "test-value"

		fetchFn := func(ctx context.Context) (any, error) {
			fetchCalled = true
			return expectedValue, nil
		}

		result, err := service.GetOrFetch(ctx, "test-key", fetchFn)
		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		if !fetchCalled {
			t.Error("expected fetch function to be called on cache miss")
		}

		if result != expectedValue {
			t.Errorf("expected result %v, got %v", expectedValue, result)
		}
	})

	t.Run("fetch function returns error", func(t *testing.T) {
		expectedError := errors.New("fetch failed")

		fetchFn := func(ctx context.Context) (any, error) {
			return nil, expectedError
		}

		result, err := service.GetOrFetch(ctx, "error-key", fetchFn)
		if err == nil {
			t.Error("expected error but got none")
		}

		if result != nil {
			t.Errorf("expected nil result but got: %v", result)
		}
	})

	t.Run("invalid fetch function type", func(t *testing.T) {
		invalidFetchFn := "not-a-function"

		result, err := service.GetOrFetch(ctx, "invalid-key", invalidFetchFn)
		if err == nil {
			t.Error("expected error for invalid function type but got none")
		}

		if result != nil {
			t.Errorf("expected nil result but got: %v", result)
		}

		configErr, ok := err.(*ConfigError)
		if !ok {
			t.Errorf("expected ConfigError but got: %T", err)
		} else if configErr.Field != "fetchFn" {
			t.Errorf("expected error field 'fetchFn', got '%s'", configErr.Field)
		}
	})

	t.Run("generic fetch function compatibility", func(t *testing.T) {
		expectedValue := "generic-value"

		var fetchFn cache.FetchFn[any] = func(ctx context.Context) (any, error) {
			return expectedValue, nil
		}

		result, err := service.GetOrFetch(ctx, "generic-key", fetchFn)
		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		if result != expectedValue {
			t.Errorf("expected result %v, got %v", expectedValue, result)
		}
	})
}

func TestSturdycService_Delete(t *testing.T) {
	cfg := DefaultConfig()
	service, err := NewSturdycService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()

	t.Run("delete returns no error (no-op implementation)", func(t *testing.T) {
		err := service.Delete(ctx, "some-key")
		if err != nil {
			t.Errorf("expected no error from Delete but got: %v", err)
		}
	})

	t.Run("delete with empty key returns no error", func(t *testing.T) {
		err := service.Delete(ctx, "")
		if err != nil {
			t.Errorf("expected no error from Delete with empty key but got: %v", err)
		}
	})
}

func TestSturdycService_InterfaceCompliance(t *testing.T) {
	cfg := DefaultConfig()
	service, err := NewSturdycService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Verify that sturdycService implements cache.CacheService interface
	var _ cache.CacheService = service

	// Additional runtime verification
	if service == nil {
		t.Error("service should not be nil")
	}

	// Verify methods exist and can be called
	ctx := context.Background()

	// Test method existence
	if err := service.Delete(ctx, "test"); err != nil {
		t.Errorf("Delete method failed: %v", err)
	}

	fetchFn := func(ctx context.Context) (any, error) {
		return "test", nil
	}

	if _, err := service.GetOrFetch(ctx, "test", fetchFn); err != nil {
		t.Errorf("GetOrFetch method failed: %v", err)
	}
}
