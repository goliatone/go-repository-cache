package cache

import (
	"context"
	"errors"
	"testing"
)

func TestCacheService_Contract(t *testing.T) {
	t.Skip("pending real adapter implementation")
	// Contract test skeleton for CacheService interface
	// This will be implemented when the sturdyc adapter is ready
}

func TestKeySerializer_Contract(t *testing.T) {
	t.Skip("pending real implementation")
	// Contract test skeleton for KeySerializer interface
	// This will be implemented when default serializer is ready
}

// mockCacheService for testing GetOrFetch function
type mockCacheService struct {
	result any
	err    error
}

func (m *mockCacheService) GetOrFetch(ctx context.Context, key string, fetchFn any) (any, error) {
	return m.result, m.err
}

func (m *mockCacheService) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCacheService) DeleteByPrefix(ctx context.Context, prefix string) error {
	return nil
}

func (m *mockCacheService) InvalidateKeys(ctx context.Context, keys []string) error {
	return nil
}

func TestGetOrFetch_NilInterfacePanic(t *testing.T) {
	// This test reproduces the panic when result is nil interface
	mock := &mockCacheService{
		result: nil, // This creates a nil interface{} which will cause panic
		err:    nil,
	}

	// Define an interface type for T
	type SomeInterface interface {
		DoSomething() string
	}

	// This should not panic - it should return zero value of SomeInterface (which is nil)
	result, err := GetOrFetch[SomeInterface](context.Background(), mock, "test-key", func(ctx context.Context) (SomeInterface, error) {
		return nil, nil
	})

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result but got: %v", result)
	}
}

func TestGetOrFetch_NilPointerNoPanic(t *testing.T) {
	// This test verifies that nil pointers work correctly
	mock := &mockCacheService{
		result: (*string)(nil), // This is a typed nil, should work
		err:    nil,
	}

	result, err := GetOrFetch[*string](context.Background(), mock, "test-key", func(ctx context.Context) (*string, error) {
		return nil, nil
	})

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result but got: %v", result)
	}
}

func TestGetOrFetch_TypeAssertionFailure(t *testing.T) {
	// This test verifies graceful handling when type assertion fails
	// (This should not happen in normal operation, but provides safety)
	mock := &mockCacheService{
		result: "wrong-type", // string instead of expected int
		err:    nil,
	}

	result, err := GetOrFetch[int](context.Background(), mock, "test-key", func(ctx context.Context) (int, error) {
		return 42, nil
	})

	if !errors.Is(err, ErrInvalidResultType) {
		t.Errorf("expected ErrInvalidResultType but got: %v", err)
	}

	if result != 0 {
		t.Errorf("expected zero value (0) but got: %v", result)
	}
}

func TestGetOrFetch_ValidResult(t *testing.T) {
	// This test verifies the happy path works correctly
	expectedValue := "test-value"
	mock := &mockCacheService{
		result: expectedValue,
		err:    nil,
	}

	result, err := GetOrFetch[string](context.Background(), mock, "test-key", func(ctx context.Context) (string, error) {
		return expectedValue, nil
	})

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if result != expectedValue {
		t.Errorf("expected '%s' but got: '%s'", expectedValue, result)
	}
}
