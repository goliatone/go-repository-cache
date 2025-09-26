package repositorycache

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	repository "github.com/goliatone/go-repository-bun"
	"github.com/goliatone/go-repository-cache/cache"
	"github.com/uptrace/bun"
)

// Interface assertion to ensure CachedRepository implements Repository[T]
var _ repository.Repository[any] = (*CachedRepository[any])(nil)

// listResult wraps the tuple result from List operations for caching
type listResult[T any] struct {
	Records []T `json:"records"`
	Total   int `json:"total"`
}

// CachedRepository decorates a base repository with caching functionality
type CachedRepository[T any] struct {
	base          repository.Repository[T]
	cache         cache.CacheService
	keySerializer cache.KeySerializer
	keyRegistry   *sync.Map // Track active cache keys for invalidation
}

// New creates a new CachedRepository that wraps the base repository with caching
func New[T any](base repository.Repository[T], cacheService cache.CacheService, keySerializer cache.KeySerializer) *CachedRepository[T] {
	return &CachedRepository[T]{
		base:          base,
		cache:         cacheService,
		keySerializer: keySerializer,
		keyRegistry:   &sync.Map{},
	}
}

// Get retrieves a single record using the provided criteria, with caching
func (c *CachedRepository[T]) Get(ctx context.Context, criteria ...repository.SelectCriteria) (T, error) {
	key := c.keySerializer.SerializeKey("Get", criteria)
	c.trackKey(key)
	return cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (T, error) {
		return c.base.Get(ctx, criteria...)
	})
}

// GetByID retrieves a record by ID with optional criteria, with caching
func (c *CachedRepository[T]) GetByID(ctx context.Context, id string, criteria ...repository.SelectCriteria) (T, error) {
	key := c.keySerializer.SerializeKey("GetByID", id, criteria)
	c.trackKey(key)
	return cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (T, error) {
		return c.base.GetByID(ctx, id, criteria...)
	})
}

// List retrieves multiple records using the provided criteria, with caching
func (c *CachedRepository[T]) List(ctx context.Context, criteria ...repository.SelectCriteria) ([]T, int, error) {
	key := c.keySerializer.SerializeKey("List", criteria)
	c.trackKey(key)
	res, err := cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (listResult[T], error) {
		records, total, err := c.base.List(ctx, criteria...)
		return listResult[T]{Records: records, Total: total}, err
	})
	if err != nil {
		return nil, 0, err
	}
	return res.Records, res.Total, nil
}

// Count returns the number of records matching the criteria, with caching
func (c *CachedRepository[T]) Count(ctx context.Context, criteria ...repository.SelectCriteria) (int, error) {
	key := c.keySerializer.SerializeKey("Count", criteria)
	c.trackKey(key)
	return cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (int, error) {
		return c.base.Count(ctx, criteria...)
	})
}

// GetByIdentifier retrieves a record by identifier with optional criteria, with caching
func (c *CachedRepository[T]) GetByIdentifier(ctx context.Context, identifier string, criteria ...repository.SelectCriteria) (T, error) {
	key := c.keySerializer.SerializeKey("GetByIdentifier", identifier, criteria)
	c.trackKey(key)
	return cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (T, error) {
		return c.base.GetByIdentifier(ctx, identifier, criteria...)
	})
}

// Create creates a new record. Write operations pass through to base repository
func (c *CachedRepository[T]) Create(ctx context.Context, record T, criteria ...repository.InsertCriteria) (T, error) {
	result, err := c.base.Create(ctx, record, criteria...)
	if err == nil {
		c.invalidateAfterCreate(ctx)
	}
	return result, err
}

// CreateTx creates a new record within a transaction
func (c *CachedRepository[T]) CreateTx(ctx context.Context, tx bun.IDB, record T, criteria ...repository.InsertCriteria) (T, error) {
	result, err := c.base.CreateTx(ctx, tx, record, criteria...)
	if err == nil {
		c.invalidateAfterCreate(ctx)
	}
	return result, err
}

// CreateMany creates multiple records
func (c *CachedRepository[T]) CreateMany(ctx context.Context, records []T, criteria ...repository.InsertCriteria) ([]T, error) {
	result, err := c.base.CreateMany(ctx, records, criteria...)
	if err == nil {
		c.invalidateAfterBulkCreate(ctx)
	}
	return result, err
}

// CreateManyTx creates multiple records within a transaction
func (c *CachedRepository[T]) CreateManyTx(ctx context.Context, tx bun.IDB, records []T, criteria ...repository.InsertCriteria) ([]T, error) {
	result, err := c.base.CreateManyTx(ctx, tx, records, criteria...)
	if err == nil {
		c.invalidateAfterBulkCreate(ctx)
	}
	return result, err
}

// GetOrCreate gets a record or creates it if it doesn't exist
func (c *CachedRepository[T]) GetOrCreate(ctx context.Context, record T) (T, error) {
	result, err := c.base.GetOrCreate(ctx, record)
	if err == nil {
		// GetOrCreate may have created a new record, so invalidate create-related caches
		c.invalidateAfterCreate(ctx)
	}
	return result, err
}

// GetOrCreateTx gets a record or creates it if it doesn't exist within a transaction
func (c *CachedRepository[T]) GetOrCreateTx(ctx context.Context, tx bun.IDB, record T) (T, error) {
	result, err := c.base.GetOrCreateTx(ctx, tx, record)
	if err == nil {
		c.invalidateAfterCreate(ctx)
	}
	return result, err
}

// Update updates a record
func (c *CachedRepository[T]) Update(ctx context.Context, record T, criteria ...repository.UpdateCriteria) (T, error) {
	result, err := c.base.Update(ctx, record, criteria...)
	if err == nil {
		c.invalidateAfterUpdate(ctx, result)
	}
	return result, err
}

// UpdateTx updates a record within a transaction
func (c *CachedRepository[T]) UpdateTx(ctx context.Context, tx bun.IDB, record T, criteria ...repository.UpdateCriteria) (T, error) {
	result, err := c.base.UpdateTx(ctx, tx, record, criteria...)
	if err == nil {
		c.invalidateAfterUpdate(ctx, result)
	}
	return result, err
}

// UpdateMany updates multiple records
func (c *CachedRepository[T]) UpdateMany(ctx context.Context, records []T, criteria ...repository.UpdateCriteria) ([]T, error) {
	result, err := c.base.UpdateMany(ctx, records, criteria...)
	if err == nil {
		c.invalidateAfterBulkUpdate(ctx, result)
	}
	return result, err
}

// UpdateManyTx updates multiple records within a transaction
func (c *CachedRepository[T]) UpdateManyTx(ctx context.Context, tx bun.IDB, records []T, criteria ...repository.UpdateCriteria) ([]T, error) {
	result, err := c.base.UpdateManyTx(ctx, tx, records, criteria...)
	if err == nil {
		c.invalidateAfterBulkUpdate(ctx, result)
	}
	return result, err
}

// Upsert inserts or updates a record
func (c *CachedRepository[T]) Upsert(ctx context.Context, record T, criteria ...repository.UpdateCriteria) (T, error) {
	result, err := c.base.Upsert(ctx, record, criteria...)
	if err == nil {
		// Upsert can either insert or update, so we need to invalidate like an update
		c.invalidateAfterUpdate(ctx, result)
	}
	return result, err
}

// UpsertTx inserts or updates a record within a transaction
func (c *CachedRepository[T]) UpsertTx(ctx context.Context, tx bun.IDB, record T, criteria ...repository.UpdateCriteria) (T, error) {
	result, err := c.base.UpsertTx(ctx, tx, record, criteria...)
	if err == nil {
		c.invalidateAfterUpdate(ctx, result)
	}
	return result, err
}

// UpsertMany inserts or updates multiple records
func (c *CachedRepository[T]) UpsertMany(ctx context.Context, records []T, criteria ...repository.UpdateCriteria) ([]T, error) {
	result, err := c.base.UpsertMany(ctx, records, criteria...)
	if err == nil {
		c.invalidateAfterBulkUpdate(ctx, result)
	}
	return result, err
}

// UpsertManyTx inserts or updates multiple records within a transaction
func (c *CachedRepository[T]) UpsertManyTx(ctx context.Context, tx bun.IDB, records []T, criteria ...repository.UpdateCriteria) ([]T, error) {
	result, err := c.base.UpsertManyTx(ctx, tx, records, criteria...)
	if err == nil {
		c.invalidateAfterBulkUpdate(ctx, result)
	}
	return result, err
}

// Delete deletes a record
func (c *CachedRepository[T]) Delete(ctx context.Context, record T) error {
	err := c.base.Delete(ctx, record)
	if err == nil {
		c.invalidateAfterDelete(ctx, record)
	}
	return err
}

// DeleteTx deletes a record within a transaction
func (c *CachedRepository[T]) DeleteTx(ctx context.Context, tx bun.IDB, record T) error {
	err := c.base.DeleteTx(ctx, tx, record)
	if err == nil {
		c.invalidateAfterDelete(ctx, record)
	}
	return err
}

// DeleteMany deletes multiple records based on criteria
func (c *CachedRepository[T]) DeleteMany(ctx context.Context, criteria ...repository.DeleteCriteria) error {
	err := c.base.DeleteMany(ctx, criteria...)
	if err == nil {
		// Since we don't have the actual records, invalidate all relevant caches
		c.invalidateAfterCriteriaOperation(ctx)
	}
	return err
}

// DeleteManyTx deletes multiple records based on criteria within a transaction
func (c *CachedRepository[T]) DeleteManyTx(ctx context.Context, tx bun.IDB, criteria ...repository.DeleteCriteria) error {
	err := c.base.DeleteManyTx(ctx, tx, criteria...)
	if err == nil {
		c.invalidateAfterCriteriaOperation(ctx)
	}
	return err
}

// DeleteWhere deletes records based on criteria
func (c *CachedRepository[T]) DeleteWhere(ctx context.Context, criteria ...repository.DeleteCriteria) error {
	err := c.base.DeleteWhere(ctx, criteria...)
	if err == nil {
		// Since we don't have the actual records, invalidate all relevant caches
		c.invalidateAfterCriteriaOperation(ctx)
	}
	return err
}

// DeleteWhereTx deletes records based on criteria within a transaction
func (c *CachedRepository[T]) DeleteWhereTx(ctx context.Context, tx bun.IDB, criteria ...repository.DeleteCriteria) error {
	err := c.base.DeleteWhereTx(ctx, tx, criteria...)
	if err == nil {
		c.invalidateAfterCriteriaOperation(ctx)
	}
	return err
}

// ForceDelete force deletes a record (bypassing soft delete)
func (c *CachedRepository[T]) ForceDelete(ctx context.Context, record T) error {
	err := c.base.ForceDelete(ctx, record)
	if err == nil {
		c.invalidateAfterDelete(ctx, record)
	}
	return err
}

// ForceDeleteTx force deletes a record within a transaction (bypassing soft delete)
func (c *CachedRepository[T]) ForceDeleteTx(ctx context.Context, tx bun.IDB, record T) error {
	err := c.base.ForceDeleteTx(ctx, tx, record)
	if err == nil {
		c.invalidateAfterDelete(ctx, record)
	}
	return err
}

// GetTx retrieves a single record using the provided criteria within a transaction
func (c *CachedRepository[T]) GetTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) (T, error) {
	return c.base.GetTx(ctx, tx, criteria...)
}

// GetByIDTx retrieves a record by ID with optional criteria within a transaction
func (c *CachedRepository[T]) GetByIDTx(ctx context.Context, tx bun.IDB, id string, criteria ...repository.SelectCriteria) (T, error) {
	return c.base.GetByIDTx(ctx, tx, id, criteria...)
}

// ListTx retrieves multiple records using the provided criteria within a transaction
func (c *CachedRepository[T]) ListTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) ([]T, int, error) {
	return c.base.ListTx(ctx, tx, criteria...)
}

// CountTx returns the number of records matching the criteria within a transaction
func (c *CachedRepository[T]) CountTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) (int, error) {
	return c.base.CountTx(ctx, tx, criteria...)
}

// GetByIdentifierTx retrieves a record by identifier with optional criteria within a transaction
func (c *CachedRepository[T]) GetByIdentifierTx(ctx context.Context, tx bun.IDB, identifier string, criteria ...repository.SelectCriteria) (T, error) {
	return c.base.GetByIdentifierTx(ctx, tx, identifier, criteria...)
}

// Raw executes a raw SQL query and returns the results
func (c *CachedRepository[T]) Raw(ctx context.Context, sql string, args ...any) ([]T, error) {
	return c.base.Raw(ctx, sql, args...)
}

// RawTx executes a raw SQL query within a transaction and returns the results
func (c *CachedRepository[T]) RawTx(ctx context.Context, tx bun.IDB, sql string, args ...any) ([]T, error) {
	return c.base.RawTx(ctx, tx, sql, args...)
}

// Handlers returns the model handlers from the base repository
func (c *CachedRepository[T]) Handlers() repository.ModelHandlers[T] {
	return c.base.Handlers()
}

// trackKey registers a cache key in the key registry for later invalidation
func (c *CachedRepository[T]) trackKey(key string) {
	c.keyRegistry.Store(key, struct{}{})
}

// invalidateByPrefix removes all cached keys that start with the given prefix
func (c *CachedRepository[T]) invalidateByPrefix(ctx context.Context, prefix string) error {
	var keysToDelete []string
	c.keyRegistry.Range(func(k, v any) bool {
		if key, ok := k.(string); ok && strings.HasPrefix(key, prefix) {
			keysToDelete = append(keysToDelete, key)
		}
		return true
	})

	for _, key := range keysToDelete {
		if err := c.cache.Delete(ctx, key); err != nil {
			// Log error but continue with other deletions
			// In a real implementation, you might want to use a proper logger
		}
		c.keyRegistry.Delete(key)
	}
	return nil
}

// extractID attempts to extract an ID field from a record using reflection
func (c *CachedRepository[T]) extractID(record T) (string, error) {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Look for common ID field names
	for _, fieldName := range []string{"ID", "Id", "id"} {
		field := v.FieldByName(fieldName)
		if field.IsValid() && field.CanInterface() {
			return fmt.Sprintf("%v", field.Interface()), nil
		}
	}
	return "", fmt.Errorf("no ID field found in record")
}

// extractIdentifier attempts to extract an identifier field from a record using reflection
func (c *CachedRepository[T]) extractIdentifier(record T) (string, error) {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Look for common identifier field names
	for _, fieldName := range []string{"Identifier", "identifier", "Name", "name", "Code", "code"} {
		field := v.FieldByName(fieldName)
		if field.IsValid() && field.CanInterface() {
			return fmt.Sprintf("%v", field.Interface()), nil
		}
	}
	return "", fmt.Errorf("no identifier field found in record")
}

// invalidateAfterCreate invalidates query result caches after create operations
func (c *CachedRepository[T]) invalidateAfterCreate(ctx context.Context) error {
	// Invalidate all List and Count caches since new records affect pagination and totals
	if err := c.invalidateByPrefix(ctx, "List"); err != nil {
		return err
	}
	return c.invalidateByPrefix(ctx, "Count")
}

// invalidateAfterUpdate invalidates all relevant caches after update operations
func (c *CachedRepository[T]) invalidateAfterUpdate(ctx context.Context, record T) error {
	// Try to invalidate specific ID-based cache
	if id, err := c.extractID(record); err == nil {
		// Invalidate all GetByID variations for this ID
		c.invalidateByPrefix(ctx, fmt.Sprintf("GetByID:%s", id))
	}

	// Try to invalidate specific identifier-based cache
	if identifier, err := c.extractIdentifier(record); err == nil {
		// Invalidate all GetByIdentifier variations for this identifier
		c.invalidateByPrefix(ctx, fmt.Sprintf("GetByIdentifier:%s", identifier))
	}

	// Invalidate all query result caches (List/Count/Get with criteria)
	c.invalidateByPrefix(ctx, "List")
	c.invalidateByPrefix(ctx, "Count")
	c.invalidateByPrefix(ctx, "Get")

	return nil
}

// invalidateAfterDelete invalidates all relevant caches after delete operations
func (c *CachedRepository[T]) invalidateAfterDelete(ctx context.Context, record T) error {
	// Same logic as update - delete affects the same caches
	return c.invalidateAfterUpdate(ctx, record)
}

// invalidateAfterBulkUpdate invalidates caches after bulk update operations
func (c *CachedRepository[T]) invalidateAfterBulkUpdate(ctx context.Context, records []T) error {
	for _, record := range records {
		if err := c.invalidateAfterUpdate(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

// invalidateAfterBulkDelete invalidates caches after bulk delete operations
func (c *CachedRepository[T]) invalidateAfterBulkDelete(ctx context.Context, records []T) error {
	for _, record := range records {
		if err := c.invalidateAfterDelete(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

// invalidateAfterBulkCreate invalidates caches after bulk create operations
func (c *CachedRepository[T]) invalidateAfterBulkCreate(ctx context.Context) error {
	// Bulk creates affect the same caches as single creates (List and Count)
	return c.invalidateAfterCreate(ctx)
}

// invalidateAfterCriteriaOperation invalidates caches after operations that use criteria instead of records
func (c *CachedRepository[T]) invalidateAfterCriteriaOperation(ctx context.Context) error {
	// For operations like DeleteMany where we don't have the actual records,
	// we must invalidate all relevant caches since we can't target specific keys
	c.invalidateByPrefix(ctx, "GetByID")
	c.invalidateByPrefix(ctx, "GetByIdentifier")
	c.invalidateByPrefix(ctx, "List")
	c.invalidateByPrefix(ctx, "Count")
	c.invalidateByPrefix(ctx, "Get")
	return nil
}
