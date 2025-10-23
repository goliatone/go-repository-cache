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
	base            repository.Repository[T]
	cache           cache.CacheService
	keySerializer   cache.KeySerializer
	namespace       string
	identifiers     []string
	scopeDefaults   repository.ScopeDefaults
	scopeDefaultsMu sync.RWMutex
}

func (c *CachedRepository[T]) setScopeDefaults(defaults repository.ScopeDefaults) {
	c.scopeDefaultsMu.Lock()
	defer c.scopeDefaultsMu.Unlock()
	c.scopeDefaults = repository.CloneScopeDefaults(defaults)
}

func (c *CachedRepository[T]) currentScopeDefaults() repository.ScopeDefaults {
	c.scopeDefaultsMu.RLock()
	defer c.scopeDefaultsMu.RUnlock()
	return repository.CloneScopeDefaults(c.scopeDefaults)
}

func (c *CachedRepository[T]) scopeSignature(ctx context.Context, op repository.ScopeOperation) repository.ScopeState {
	defaults := c.currentScopeDefaults()
	return repository.ResolveScopeState(ctx, defaults, op)
}

func toAnySlice[T any](items []T) []any {
	if len(items) == 0 {
		return nil
	}
	args := make([]any, len(items))
	for i, item := range items {
		args[i] = item
	}
	return args
}

// New creates a new CachedRepository that wraps the base repository with caching
func New[T any](base repository.Repository[T], cacheService cache.CacheService, keySerializer cache.KeySerializer) *CachedRepository[T] {
	return newCachedRepository(base, cacheService, keySerializer, nil)
}

// NewWithIdentifierFields creates a CachedRepository with custom identifier field names.
// Field names must match the struct field names returned by the repository handlers.
func NewWithIdentifierFields[T any](base repository.Repository[T], cacheService cache.CacheService, keySerializer cache.KeySerializer, identifierFields ...string) *CachedRepository[T] {
	return newCachedRepository(base, cacheService, keySerializer, identifierFields)
}

// Get retrieves a single record using the provided criteria, with caching
func (c *CachedRepository[T]) Get(ctx context.Context, criteria ...repository.SelectCriteria) (T, error) {
	signature := c.scopeSignature(ctx, repository.ScopeOperationSelect)
	args := toAnySlice(criteria)
	if !signature.IsZero() {
		args = append([]any{signature}, args...)
	}
	key := c.key("Get", args...)
	return cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (T, error) {
		return c.base.Get(ctx, criteria...)
	})
}

// GetByID retrieves a record by ID with optional criteria, with caching
func (c *CachedRepository[T]) GetByID(ctx context.Context, id string, criteria ...repository.SelectCriteria) (T, error) {
	signature := c.scopeSignature(ctx, repository.ScopeOperationSelect)
	args := []any{id}
	if !signature.IsZero() {
		args = append(args, signature)
	}
	args = append(args, toAnySlice(criteria)...)
	key := c.key("GetByID", args...)
	return cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (T, error) {
		return c.base.GetByID(ctx, id, criteria...)
	})
}

// List retrieves multiple records using the provided criteria, with caching
func (c *CachedRepository[T]) List(ctx context.Context, criteria ...repository.SelectCriteria) ([]T, int, error) {
	signature := c.scopeSignature(ctx, repository.ScopeOperationSelect)
	args := toAnySlice(criteria)
	if !signature.IsZero() {
		args = append([]any{signature}, args...)
	}
	key := c.key("List", args...)
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
	signature := c.scopeSignature(ctx, repository.ScopeOperationSelect)
	args := toAnySlice(criteria)
	if !signature.IsZero() {
		args = append([]any{signature}, args...)
	}
	key := c.key("Count", args...)
	return cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (int, error) {
		return c.base.Count(ctx, criteria...)
	})
}

// GetByIdentifier retrieves a record by identifier with optional criteria, with caching
func (c *CachedRepository[T]) GetByIdentifier(ctx context.Context, identifier string, criteria ...repository.SelectCriteria) (T, error) {
	signature := c.scopeSignature(ctx, repository.ScopeOperationSelect)
	args := []any{identifier}
	if !signature.IsZero() {
		args = append(args, signature)
	}
	args = append(args, toAnySlice(criteria)...)
	key := c.key("GetByIdentifier", args...)
	return cache.GetOrFetch(ctx, c.cache, key, func(ctx context.Context) (T, error) {
		return c.base.GetByIdentifier(ctx, identifier, criteria...)
	})
}

// Create creates a new record. Write operations pass through to base repository
func (c *CachedRepository[T]) Create(ctx context.Context, record T, criteria ...repository.InsertCriteria) (T, error) {
	result, err := c.base.Create(ctx, record, criteria...)
	if err == nil {
		c.invalidateAfterCreate(ctx, result)
	}
	return result, err
}

// CreateTx creates a new record within a transaction
func (c *CachedRepository[T]) CreateTx(ctx context.Context, tx bun.IDB, record T, criteria ...repository.InsertCriteria) (T, error) {
	result, err := c.base.CreateTx(ctx, tx, record, criteria...)
	if err == nil {
		c.invalidateAfterCreate(ctx, result)
	}
	return result, err
}

// CreateMany creates multiple records
func (c *CachedRepository[T]) CreateMany(ctx context.Context, records []T, criteria ...repository.InsertCriteria) ([]T, error) {
	result, err := c.base.CreateMany(ctx, records, criteria...)
	if err == nil {
		c.invalidateAfterBulkCreate(ctx, result)
	}
	return result, err
}

// CreateManyTx creates multiple records within a transaction
func (c *CachedRepository[T]) CreateManyTx(ctx context.Context, tx bun.IDB, records []T, criteria ...repository.InsertCriteria) ([]T, error) {
	result, err := c.base.CreateManyTx(ctx, tx, records, criteria...)
	if err == nil {
		c.invalidateAfterBulkCreate(ctx, result)
	}
	return result, err
}

// GetOrCreate gets a record or creates it if it doesn't exist
func (c *CachedRepository[T]) GetOrCreate(ctx context.Context, record T) (T, error) {
	result, err := c.base.GetOrCreate(ctx, record)
	if err == nil {
		// GetOrCreate may have created a new record, so invalidate create related caches
		c.invalidateAfterCreate(ctx, result)
	}
	return result, err
}

// GetOrCreateTx gets a record or creates it if it doesn't exist within a transaction
func (c *CachedRepository[T]) GetOrCreateTx(ctx context.Context, tx bun.IDB, record T) (T, error) {
	result, err := c.base.GetOrCreateTx(ctx, tx, record)
	if err == nil {
		c.invalidateAfterCreate(ctx, result)
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

func (c *CachedRepository[T]) RegisterScope(name string, scope repository.ScopeDefinition) {
	c.base.RegisterScope(name, scope)
}

func (c *CachedRepository[T]) SetScopeDefaults(defaults repository.ScopeDefaults) {
	c.base.SetScopeDefaults(defaults)
	c.setScopeDefaults(defaults)
}

func (c *CachedRepository[T]) GetScopeDefaults() repository.ScopeDefaults {
	return c.currentScopeDefaults()
}

// extractID attempts to extract an ID field from a record using reflection
func (c *CachedRepository[T]) extractID(record T) (string, error) {
	v, err := structValue(record)
	if err != nil {
		return "", err
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
func (c *CachedRepository[T]) extractIdentifierValues(record T) ([]string, error) {
	if len(c.identifiers) == 0 {
		return nil, fmt.Errorf("no identifier fields configured")
	}

	v, err := structValue(record)
	if err != nil {
		return nil, err
	}

	values := make([]string, 0, len(c.identifiers))
	for _, fieldName := range c.identifiers {
		field := v.FieldByName(fieldName)
		if !field.IsValid() || !field.CanInterface() {
			continue
		}
		if val, ok := valueToString(field); ok {
			values = append(values, val)
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no identifier values found")
	}

	return values, nil
}

func (c *CachedRepository[T]) invalidateRecordCaches(ctx context.Context, record T) {
	if id, err := c.extractID(record); err == nil && id != "" {
		c.deleteByPrefix(ctx, c.methodPrefix("GetByID", id))
	}

	if identifiers, err := c.extractIdentifierValues(record); err == nil {
		for _, identifier := range identifiers {
			if identifier == "" {
				continue
			}
			c.deleteByPrefix(ctx, c.methodPrefix("GetByIdentifier", identifier))
		}
	} else {
		c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("GetByIdentifier"))
	}
}

// invalidateAfterCreate invalidates caches after create operations
func (c *CachedRepository[T]) invalidateAfterCreate(ctx context.Context, records ...T) error {
	for _, record := range records {
		c.invalidateRecordCaches(ctx, record)
	}

	c.invalidateGetCaches(ctx)
	c.deleteKey(ctx, "List")
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("List"))
	c.deleteKey(ctx, "Count")
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("Count"))

	return nil
}

// invalidateAfterUpdate invalidates all relevant caches after update operations
func (c *CachedRepository[T]) invalidateAfterUpdate(ctx context.Context, record T) error {
	c.invalidateRecordCaches(ctx, record)

	// Invalidate all query result caches (List/Count/Get with criteria)
	c.invalidateGetCaches(ctx)
	c.deleteKey(ctx, "List")
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("List"))
	c.deleteKey(ctx, "Count")
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("Count"))

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

// invalidateAfterBulkCreate invalidates caches after bulk create operations
func (c *CachedRepository[T]) invalidateAfterBulkCreate(ctx context.Context, records []T) error {
	return c.invalidateAfterCreate(ctx, records...)
}

// invalidateAfterCriteriaOperation invalidates caches after operations that use criteria instead of records
func (c *CachedRepository[T]) invalidateAfterCriteriaOperation(ctx context.Context) error {
	// For operations like DeleteMany where we don't have the actual records,
	// we must invalidate all relevant caches since we can't target specific keys
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("GetByID"))
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("GetByIdentifier"))
	c.deleteKey(ctx, "List")
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("List"))
	c.deleteKey(ctx, "Count")
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("Count"))
	c.invalidateGetCaches(ctx)
	return nil
}

func structValue(record any) (reflect.Value, error) {
	v := reflect.ValueOf(record)
	if !v.IsValid() {
		return reflect.Value{}, fmt.Errorf("record is invalid")
	}

	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, fmt.Errorf("record interface is nil")
		}
		v = v.Elem()
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}, fmt.Errorf("record pointer is nil")
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("record type %s is not a struct", v.Kind())
	}

	return v, nil
}

func valueToString(v reflect.Value) (string, bool) {
	value := v
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return "", false
		}
		value = value.Elem()
	}

	if !value.CanInterface() {
		return "", false
	}

	interf := value.Interface()
	switch val := interf.(type) {
	case fmt.Stringer:
		return val.String(), true
	case string:
		if val == "" {
			return "", false
		}
		return val, true
	default:
		str := fmt.Sprintf("%v", interf)
		if str == "" || str == "<nil>" {
			return "", false
		}
		return str, true
	}
}

func newCachedRepository[T any](base repository.Repository[T], cache cache.CacheService, serializer cache.KeySerializer, identifierFields []string) *CachedRepository[T] {
	repo := &CachedRepository[T]{
		base:          base,
		cache:         cache,
		keySerializer: serializer,
		namespace:     deriveNamespace(base),
	}
	repo.identifiers = repo.resolveIdentifierFields(identifierFields)
	repo.setScopeDefaults(base.GetScopeDefaults())
	return repo
}

func deriveNamespace[T any](_ repository.Repository[T]) string {
	var sample T
	typ := reflect.TypeOf(sample)
	if typ == nil {
		var ptr *T
		typ = reflect.TypeOf(ptr)
	}
	if typ == nil {
		return "unknown"
	}

	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	name := typ.Name()
	if name == "" {
		name = typ.String()
		if idx := strings.LastIndex(name, "."); idx != -1 {
			name = name[idx+1:]
		}
	}

	return toSnake(name)
}

func (c *CachedRepository[T]) resolveIdentifierFields(explicit []string) []string {
	if len(explicit) > 0 {
		return dedupeStrings(explicit)
	}
	derived := deriveIdentifierFields(c.base)
	return dedupeStrings(derived)
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func deriveIdentifierFields[T any](base repository.Repository[T]) []string {
	handlers, ok := safeHandlers(base)
	if !ok || handlers.NewRecord == nil {
		return nil
	}

	record := handlers.NewRecord()
	meta := repository.GenerateModelMeta(record)

	var fields []string
	for _, field := range meta.Fields {
		if field.IsUnique {
			fields = append(fields, field.StructName)
		}
	}

	return fields
}

func safeHandlers[T any](base repository.Repository[T]) (repository.ModelHandlers[T], bool) {
	var (
		h  repository.ModelHandlers[T]
		ok bool
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				ok = false
			}
		}()
		h = base.Handlers()
		ok = true
	}()
	return h, ok
}

func (c *CachedRepository[T]) key(method string, args ...any) string {
	methodKey := c.methodKey(method)
	return c.keySerializer.SerializeKey(methodKey, args...)
}

func (c *CachedRepository[T]) methodKey(method string) string {
	return strings.Join([]string{c.namespace, toSnake(method)}, cache.KeySeparator)
}

func (c *CachedRepository[T]) methodPrefix(method string, segments ...string) string {
	prefix := c.methodKey(method)
	if len(segments) == 0 {
		return prefix
	}
	return prefix + cache.KeySeparator + strings.Join(segments, cache.KeySeparator)
}

func (c *CachedRepository[T]) methodPrefixWithSeparator(method string) string {
	return c.methodKey(method) + cache.KeySeparator
}

func (c *CachedRepository[T]) deleteKey(ctx context.Context, method string, args ...any) {
	key := c.key(method, args...)
	_ = c.cache.Delete(ctx, key)
}

func (c *CachedRepository[T]) deleteByPrefix(ctx context.Context, prefix string) {
	_ = c.cache.DeleteByPrefix(ctx, prefix)
}

func (c *CachedRepository[T]) invalidateGetCaches(ctx context.Context) {
	c.deleteKey(ctx, "Get")
	c.deleteByPrefix(ctx, c.methodPrefixWithSeparator("Get"))
}
