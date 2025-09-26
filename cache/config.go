package cache

import (
	"time"

	"github.com/goliatone/go-repository-cache/internal/cacheinfra"
)

// Config exposes cache configuration options for consumers of the cache package.
type Config struct {
	Capacity             int
	NumShards            int
	TTL                  time.Duration
	EvictionPercentage   int
	EarlyRefresh         *EarlyRefreshConfig
	MissingRecordStorage bool
	EvictionInterval     time.Duration
}

// EarlyRefreshConfig mirrors the underlying sturdyc early refresh options.
type EarlyRefreshConfig struct {
	MinAsyncRefreshTime time.Duration
	MaxAsyncRefreshTime time.Duration
	SyncRefreshTime     time.Duration
	RetryBaseDelay      time.Duration
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	return convertFromInternal(cacheinfra.DefaultConfig())
}

// Validate checks whether the configuration values are valid.
func (c Config) Validate() error {
	return c.toInternal().Validate()
}

// NewCacheService constructs the default cache service implementation using the provided configuration.
func NewCacheService(cfg Config) (CacheService, error) {
	return cacheinfra.NewSturdycService(cfg.toInternal())
}

func (c Config) toInternal() cacheinfra.Config {
	var early *cacheinfra.EarlyRefreshConfig
	if c.EarlyRefresh != nil {
		early = &cacheinfra.EarlyRefreshConfig{
			MinAsyncRefreshTime: c.EarlyRefresh.MinAsyncRefreshTime,
			MaxAsyncRefreshTime: c.EarlyRefresh.MaxAsyncRefreshTime,
			SyncRefreshTime:     c.EarlyRefresh.SyncRefreshTime,
			RetryBaseDelay:      c.EarlyRefresh.RetryBaseDelay,
		}
	}

	return cacheinfra.Config{
		Capacity:             c.Capacity,
		NumShards:            c.NumShards,
		TTL:                  c.TTL,
		EvictionPercentage:   c.EvictionPercentage,
		EarlyRefresh:         early,
		MissingRecordStorage: c.MissingRecordStorage,
		EvictionInterval:     c.EvictionInterval,
	}
}

func convertFromInternal(cfg cacheinfra.Config) Config {
	var early *EarlyRefreshConfig
	if cfg.EarlyRefresh != nil {
		early = &EarlyRefreshConfig{
			MinAsyncRefreshTime: cfg.EarlyRefresh.MinAsyncRefreshTime,
			MaxAsyncRefreshTime: cfg.EarlyRefresh.MaxAsyncRefreshTime,
			SyncRefreshTime:     cfg.EarlyRefresh.SyncRefreshTime,
			RetryBaseDelay:      cfg.EarlyRefresh.RetryBaseDelay,
		}
	}

	return Config{
		Capacity:             cfg.Capacity,
		NumShards:            cfg.NumShards,
		TTL:                  cfg.TTL,
		EvictionPercentage:   cfg.EvictionPercentage,
		EarlyRefresh:         early,
		MissingRecordStorage: cfg.MissingRecordStorage,
		EvictionInterval:     cfg.EvictionInterval,
	}
}
