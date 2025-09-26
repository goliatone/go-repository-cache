package cache

import (
	"context"
	"errors"
	"fmt"
)

// KeySerializer builds a cache key from a method name + arbitrary args.
// It is responsible for producing stable keys across calls.
type KeySerializer interface {
	SerializeKey(method string, args ...any) string
}

// FetchFn is the function signature CacheService expects when fetching from the source of truth.
type FetchFn[T any] func(ctx context.Context) (T, error)

// CacheService exposes the read-through caching operations we need when decorating repositories.
// It is exported so that other packages can reuse the default serializer or provide alternate cache backends.
type CacheService interface {
	GetOrFetch(ctx context.Context, key string, fetchFn any) (any, error)
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) error
	InvalidateKeys(ctx context.Context, keys []string) error
}

// ErrInvalidResultType indicates that the underlying cache implementation returned a value
// that cannot be asserted to the requested generic type.
var ErrInvalidResultType = errors.New("cache: invalid result type")

// GetOrFetch is a type-safe wrapper function that provides generic support for CacheService.
func GetOrFetch[T any](ctx context.Context, service CacheService, key string, fetchFn FetchFn[T]) (T, error) {
	result, err := service.GetOrFetch(ctx, key, fetchFn)
	if err != nil {
		var zero T
		return zero, err
	}

	// Handle nil interface case: if result is nil, return zero value of T
	// This prevents panic when T is an interface and result is a nil interface{}
	if result == nil {
		var zero T
		return zero, nil
	}

	// Use comma-ok form to safely assert the type
	// Return an explicit error if the underlying value has the wrong type
	if typedResult, ok := result.(T); ok {
		return typedResult, nil
	}

	var zero T
	return zero, fmt.Errorf("%w (got %T)", ErrInvalidResultType, result)
}
