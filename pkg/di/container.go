package di

import (
	repository "github.com/goliatone/go-repository-bun"
	"github.com/goliatone/go-repository-cache/cache"
	"github.com/goliatone/go-repository-cache/internal/cacheinfra"
	"github.com/goliatone/go-repository-cache/repositorycache"
)

// Container provides dependency injection for cache related components.
// It manages singleton instances of cache services and key serializers,
// and provides factory methods for creating cached repositories.
type Container struct {
	cacheService  cache.CacheService
	keySerializer cache.KeySerializer
	config        cacheinfra.Config
}

// NewContainer creates a new DI container with the provided cache configuration.
// It initializes the cache service using the sturdyc adapter and sets up
// the default key serializer for consistent key generation.
func NewContainer(config cacheinfra.Config) (*Container, error) {
	// Initialize the cache service using the sturdyc adapter
	cacheService, err := cacheinfra.NewSturdycService(config)
	if err != nil {
		return nil, err
	}

	// Initialize the default key serializer
	keySerializer := cache.NewDefaultKeySerializer()

	return &Container{
		cacheService:  cacheService,
		keySerializer: keySerializer,
		config:        config,
	}, nil
}

// NewContainerWithDefaults creates a new DI container using default configuration.
// This is a convenience constructor for typical use cases where custom configuration
// is not required.
func NewContainerWithDefaults() (*Container, error) {
	return NewContainer(cacheinfra.DefaultConfig())
}

// CacheService returns the singleton cache service instance.
// This allows access to the underlying cache for advanced use cases.
func (c *Container) CacheService() cache.CacheService {
	return c.cacheService
}

// KeySerializer returns the singleton key serializer instance.
// This allows access to the key serializer for custom caching implementations.
func (c *Container) KeySerializer() cache.KeySerializer {
	return c.keySerializer
}

// Config returns a copy of the cache configuration used by this container.
// This is useful for debugging and monitoring purposes.
func (c *Container) Config() cacheinfra.Config {
	return c.config
}

// NewCachedRepository creates a new cached repository that wraps the provided base repository.
// It wires together the cache service, key serializer, and base repository to provide
// a drop-in replacement with caching capabilities.
//
// Since Go methods cannot have type parameters, this is provided as a package-level function.
// Example: NewCachedRepository[User](container, baseUserRepository)
func NewCachedRepository[T any](container *Container, base repository.Repository[T]) *repositorycache.CachedRepository[T] {
	return repositorycache.New(base, container.cacheService, container.keySerializer)
}
