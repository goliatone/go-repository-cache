package repositorycache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	repository "github.com/goliatone/go-repository-bun"
	"github.com/goliatone/go-repository-cache/cache"
	"github.com/uptrace/bun"
)

// TestUser represents a test entity
type TestUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// mockRepository is a comprehensive mock that tracks method calls for testing
type mockRepository[T any] struct {
	mu             sync.Mutex
	calls          []string
	getResult      T
	getError       error
	getByIDResult  T
	getByIDError   error
	listRecords    []T
	listTotal      int
	listError      error
	countResult    int
	countError     error
	getByIDResult2 T
	getByIDError2  error
	createResult   T
	createError    error
	updateResult   T
	updateError    error
	deleteError    error
	scopeDefaults  repository.ScopeDefaults
}

// Helper method to record method calls
func (m *mockRepository[T]) recordCall(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, method)
}

// Helper method to get recorded calls
func (m *mockRepository[T]) getCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.calls...)
}

// Helper method to clear recorded calls
func (m *mockRepository[T]) clearCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
}

// READ methods that we want to test caching for
func (m *mockRepository[T]) Get(ctx context.Context, criteria ...repository.SelectCriteria) (T, error) {
	m.recordCall("Get")
	return m.getResult, m.getError
}

func (m *mockRepository[T]) GetByID(ctx context.Context, id string, criteria ...repository.SelectCriteria) (T, error) {
	m.recordCall("GetByID")
	return m.getByIDResult, m.getByIDError
}

func (m *mockRepository[T]) List(ctx context.Context, criteria ...repository.SelectCriteria) ([]T, int, error) {
	m.recordCall("List")
	return m.listRecords, m.listTotal, m.listError
}

func (m *mockRepository[T]) Count(ctx context.Context, criteria ...repository.SelectCriteria) (int, error) {
	m.recordCall("Count")
	return m.countResult, m.countError
}

func (m *mockRepository[T]) GetByIdentifier(ctx context.Context, identifier string, criteria ...repository.SelectCriteria) (T, error) {
	m.recordCall("GetByIdentifier")
	return m.getByIDResult2, m.getByIDError2
}

// WRITE methods that we want to test delegation for
func (m *mockRepository[T]) Create(ctx context.Context, record T, criteria ...repository.InsertCriteria) (T, error) {
	m.recordCall("Create")
	return m.createResult, m.createError
}

func (m *mockRepository[T]) Update(ctx context.Context, record T, criteria ...repository.UpdateCriteria) (T, error) {
	m.recordCall("Update")
	return m.updateResult, m.updateError
}

func (m *mockRepository[T]) Delete(ctx context.Context, record T) error {
	m.recordCall("Delete")
	return m.deleteError
}

func (m *mockRepository[T]) RegisterScope(name string, scope repository.ScopeDefinition) {
	m.recordCall("RegisterScope")
}

func (m *mockRepository[T]) SetScopeDefaults(defaults repository.ScopeDefaults) {
	m.recordCall("SetScopeDefaults")
	m.scopeDefaults = repository.CloneScopeDefaults(defaults)
}

func (m *mockRepository[T]) GetScopeDefaults() repository.ScopeDefaults {
	return repository.CloneScopeDefaults(m.scopeDefaults)
}

// Other methods that panic to ensure they're not called during our tests
func (m *mockRepository[T]) Raw(ctx context.Context, sql string, args ...any) ([]T, error) {
	panic("Raw not implemented in mock - should not be called in cache tests")
}
func (m *mockRepository[T]) RawTx(ctx context.Context, tx bun.IDB, sql string, args ...any) ([]T, error) {
	panic("RawTx not implemented in mock")
}
func (m *mockRepository[T]) GetTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) (T, error) {
	panic("GetTx not implemented in mock")
}
func (m *mockRepository[T]) GetByIDTx(ctx context.Context, tx bun.IDB, id string, criteria ...repository.SelectCriteria) (T, error) {
	panic("GetByIDTx not implemented in mock")
}
func (m *mockRepository[T]) ListTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) ([]T, int, error) {
	panic("ListTx not implemented in mock")
}
func (m *mockRepository[T]) CountTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) (int, error) {
	panic("CountTx not implemented in mock")
}
func (m *mockRepository[T]) CreateTx(ctx context.Context, tx bun.IDB, record T, criteria ...repository.InsertCriteria) (T, error) {
	panic("CreateTx not implemented in mock")
}
func (m *mockRepository[T]) CreateMany(ctx context.Context, records []T, criteria ...repository.InsertCriteria) ([]T, error) {
	m.recordCall("CreateMany")
	return records, m.createError
}
func (m *mockRepository[T]) CreateManyTx(ctx context.Context, tx bun.IDB, records []T, criteria ...repository.InsertCriteria) ([]T, error) {
	panic("CreateManyTx not implemented in mock")
}
func (m *mockRepository[T]) GetOrCreate(ctx context.Context, record T) (T, error) {
	panic("GetOrCreate not implemented in mock")
}
func (m *mockRepository[T]) GetOrCreateTx(ctx context.Context, tx bun.IDB, record T) (T, error) {
	panic("GetOrCreateTx not implemented in mock")
}
func (m *mockRepository[T]) GetByIdentifierTx(ctx context.Context, tx bun.IDB, identifier string, criteria ...repository.SelectCriteria) (T, error) {
	panic("GetByIdentifierTx not implemented in mock")
}
func (m *mockRepository[T]) UpdateTx(ctx context.Context, tx bun.IDB, record T, criteria ...repository.UpdateCriteria) (T, error) {
	panic("UpdateTx not implemented in mock")
}
func (m *mockRepository[T]) UpdateMany(ctx context.Context, records []T, criteria ...repository.UpdateCriteria) ([]T, error) {
	panic("UpdateMany not implemented in mock")
}
func (m *mockRepository[T]) UpdateManyTx(ctx context.Context, tx bun.IDB, records []T, criteria ...repository.UpdateCriteria) ([]T, error) {
	panic("UpdateManyTx not implemented in mock")
}
func (m *mockRepository[T]) Upsert(ctx context.Context, record T, criteria ...repository.UpdateCriteria) (T, error) {
	panic("Upsert not implemented in mock")
}
func (m *mockRepository[T]) UpsertTx(ctx context.Context, tx bun.IDB, record T, criteria ...repository.UpdateCriteria) (T, error) {
	panic("UpsertTx not implemented in mock")
}
func (m *mockRepository[T]) UpsertMany(ctx context.Context, records []T, criteria ...repository.UpdateCriteria) ([]T, error) {
	panic("UpsertMany not implemented in mock")
}
func (m *mockRepository[T]) UpsertManyTx(ctx context.Context, tx bun.IDB, records []T, criteria ...repository.UpdateCriteria) ([]T, error) {
	panic("UpsertManyTx not implemented in mock")
}
func (m *mockRepository[T]) DeleteTx(ctx context.Context, tx bun.IDB, record T) error {
	panic("DeleteTx not implemented in mock")
}
func (m *mockRepository[T]) DeleteMany(ctx context.Context, criteria ...repository.DeleteCriteria) error {
	m.recordCall("DeleteMany")
	return m.deleteError
}
func (m *mockRepository[T]) DeleteManyTx(ctx context.Context, tx bun.IDB, criteria ...repository.DeleteCriteria) error {
	panic("DeleteManyTx not implemented in mock")
}
func (m *mockRepository[T]) DeleteWhere(ctx context.Context, criteria ...repository.DeleteCriteria) error {
	panic("DeleteWhere not implemented in mock")
}
func (m *mockRepository[T]) DeleteWhereTx(ctx context.Context, tx bun.IDB, criteria ...repository.DeleteCriteria) error {
	panic("DeleteWhereTx not implemented in mock")
}
func (m *mockRepository[T]) ForceDelete(ctx context.Context, record T) error {
	panic("ForceDelete not implemented in mock")
}
func (m *mockRepository[T]) ForceDeleteTx(ctx context.Context, tx bun.IDB, record T) error {
	panic("ForceDeleteTx not implemented in mock")
}
func (m *mockRepository[T]) Handlers() repository.ModelHandlers[T] {
	panic("Handlers not implemented in mock")
}

// mockCacheService tracks cache operations and can simulate cache hits/misses
type mockCacheService struct {
	mu      sync.Mutex
	calls   []string
	storage map[string]any
	hits    map[string]bool
	errors  map[string]error
}

func newMockCacheService() *mockCacheService {
	return &mockCacheService{
		storage: make(map[string]any),
		hits:    make(map[string]bool),
		errors:  make(map[string]error),
	}
}

// SetCacheValue pre-populates cache to simulate cache hit
func (m *mockCacheService) SetCacheValue(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storage[key] = value
	m.hits[key] = true
}

// SetCacheError simulates cache error for specific key
func (m *mockCacheService) SetCacheError(key string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[key] = err
}

func (m *mockCacheService) recordCall(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, method)
}

func (m *mockCacheService) getCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.calls...)
}

func (m *mockCacheService) GetOrFetch(ctx context.Context, key string, fetchFn any) (any, error) {
	m.recordCall(fmt.Sprintf("GetOrFetch:%s", key))

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for configured error
	if err, exists := m.errors[key]; exists {
		return nil, err
	}

	// Check for cache hit
	if value, exists := m.storage[key]; exists {
		return value, nil
	}

	// Cache miss - call fetch function
	fv := reflect.ValueOf(fetchFn)
	result := fv.Call([]reflect.Value{reflect.ValueOf(ctx)})

	if len(result) != 2 {
		return nil, errors.New("fetchFn must return (T, error)")
	}

	// Check if error is returned
	if !result[1].IsNil() {
		return nil, result[1].Interface().(error)
	}

	// Store result in cache and return
	value := result[0].Interface()
	m.storage[key] = value
	return value, nil
}

func (m *mockCacheService) Delete(ctx context.Context, key string) error {
	m.recordCall(fmt.Sprintf("Delete:%s", key))
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.storage, key)
	delete(m.hits, key)
	return nil
}

func (m *mockCacheService) DeleteByPrefix(ctx context.Context, prefix string) error {
	m.recordCall(fmt.Sprintf("DeleteByPrefix:%s", prefix))
	m.mu.Lock()
	defer m.mu.Unlock()

	for key := range m.storage {
		if len(prefix) == 0 || (len(key) >= len(prefix) && key[:len(prefix)] == prefix) {
			delete(m.storage, key)
			delete(m.hits, key)
		}
	}
	return nil
}

func (m *mockCacheService) InvalidateKeys(ctx context.Context, keys []string) error {
	m.recordCall(fmt.Sprintf("InvalidateKeys:%v", keys))
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.storage, key)
		delete(m.hits, key)
	}
	return nil
}

// trackingKeySerializer tracks serialization calls and allows customization
type trackingKeySerializer struct {
	mu    sync.Mutex
	calls []string
	keys  map[string]string // allows custom key mapping
}

func newTrackingKeySerializer() *trackingKeySerializer {
	return &trackingKeySerializer{
		keys: make(map[string]string),
	}
}

func (t *trackingKeySerializer) SetCustomKey(method string, args string, key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	cacheKey := fmt.Sprintf("%s:%s", method, args)
	t.keys[cacheKey] = key
}

func (t *trackingKeySerializer) SerializeKey(method string, args ...any) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	argStr := fmt.Sprintf("%v", args)
	cacheKey := fmt.Sprintf("%s:%s", method, argStr)
	t.calls = append(t.calls, cacheKey)

	// Return custom key if configured
	if customKey, exists := t.keys[cacheKey]; exists {
		return customKey
	}

	if len(args) == 0 {
		return method
	}

	parts := make([]string, 0, len(args)+1)
	parts = append(parts, method)
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%v", arg))
	}

	return strings.Join(parts, cache.KeySeparator)
}

func (t *trackingKeySerializer) getCalls() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.calls...)
}

func TestNew(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	cached := New(baseRepo, cacheService, keySerializer)

	if cached == nil {
		t.Fatal("New() returned nil")
	}

	if cached.base != baseRepo {
		t.Error("base repository not stored correctly")
	}

	if cached.cache == nil {
		t.Error("cache service not stored correctly")
	}

	if cached.keySerializer != keySerializer {
		t.Error("key serializer not stored correctly")
	}
}

// Test cache hit scenarios for read methods
func TestCachedReadMethods_CacheHit(t *testing.T) {
	tests := []struct {
		name          string
		setupCache    func(*mockCacheService, *CachedRepository[TestUser])
		setupRepo     func(*mockRepository[TestUser])
		testOperation func(*CachedRepository[TestUser]) error
		expectedCalls []string
	}{
		{
			name: "Get_CacheHit",
			setupCache: func(cache *mockCacheService, cached *CachedRepository[TestUser]) {
				cache.SetCacheValue(cached.key("Get"), TestUser{ID: "cached-1", Name: "Cached User"})
			},
			setupRepo: func(repo *mockRepository[TestUser]) {
				// Should not be called due to cache hit
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user, err := cached.Get(context.Background())
				if err != nil {
					return err
				}
				if user.ID != "cached-1" {
					return fmt.Errorf("expected cached user ID 'cached-1', got '%s'", user.ID)
				}
				return nil
			},
			expectedCalls: []string{}, // Base repo should not be called
		},
		{
			name: "GetByID_CacheHit",
			setupCache: func(cache *mockCacheService, cached *CachedRepository[TestUser]) {
				cache.SetCacheValue(cached.key("GetByID", "user-1"), TestUser{ID: "user-1", Name: "Cached User"})
			},
			setupRepo: func(repo *mockRepository[TestUser]) {},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user, err := cached.GetByID(context.Background(), "user-1")
				if err != nil {
					return err
				}
				if user.ID != "user-1" {
					return fmt.Errorf("expected user ID 'user-1', got '%s'", user.ID)
				}
				return nil
			},
			expectedCalls: []string{},
		},
		{
			name: "List_CacheHit",
			setupCache: func(cache *mockCacheService, cached *CachedRepository[TestUser]) {
				result := listResult[TestUser]{
					Records: []TestUser{{ID: "1", Name: "User 1"}, {ID: "2", Name: "User 2"}},
					Total:   2,
				}
				cache.SetCacheValue(cached.key("List"), result)
			},
			setupRepo: func(repo *mockRepository[TestUser]) {},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				records, total, err := cached.List(context.Background())
				if err != nil {
					return err
				}
				if len(records) != 2 || total != 2 {
					return fmt.Errorf("expected 2 records and total 2, got %d records and total %d", len(records), total)
				}
				return nil
			},
			expectedCalls: []string{},
		},
		{
			name: "Count_CacheHit",
			setupCache: func(cache *mockCacheService, cached *CachedRepository[TestUser]) {
				cache.SetCacheValue(cached.key("Count"), 42)
			},
			setupRepo: func(repo *mockRepository[TestUser]) {},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				count, err := cached.Count(context.Background())
				if err != nil {
					return err
				}
				if count != 42 {
					return fmt.Errorf("expected count 42, got %d", count)
				}
				return nil
			},
			expectedCalls: []string{},
		},
		{
			name: "GetByIdentifier_CacheHit",
			setupCache: func(cache *mockCacheService, cached *CachedRepository[TestUser]) {
				cache.SetCacheValue(cached.key("GetByIdentifier", "username123"), TestUser{ID: "user-1", Name: "User by identifier"})
			},
			setupRepo: func(repo *mockRepository[TestUser]) {},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user, err := cached.GetByIdentifier(context.Background(), "username123")
				if err != nil {
					return err
				}
				if user.Name != "User by identifier" {
					return fmt.Errorf("expected user name 'User by identifier', got '%s'", user.Name)
				}
				return nil
			},
			expectedCalls: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseRepo := &mockRepository[TestUser]{}
			cacheService := newMockCacheService()
			keySerializer := newTrackingKeySerializer()

			tt.setupRepo(baseRepo)

			cached := New(baseRepo, cacheService, keySerializer)
			tt.setupCache(cacheService, cached)

			err := tt.testOperation(cached)
			if err != nil {
				t.Fatalf("test operation failed: %v", err)
			}

			// Verify base repository was not called (cache hit)
			calls := baseRepo.getCalls()
			if len(calls) != len(tt.expectedCalls) {
				t.Errorf("expected %d base repo calls, got %d: %v", len(tt.expectedCalls), len(calls), calls)
			}
		})
	}
}

// Test cache miss scenarios for read methods
func TestCachedReadMethods_CacheMiss(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func(*mockRepository[TestUser])
		testOperation func(*CachedRepository[TestUser]) error
		expectedCalls []string
	}{
		{
			name: "Get_CacheMiss",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.getResult = TestUser{ID: "fetched-1", Name: "Fetched User"}
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user, err := cached.Get(context.Background())
				if err != nil {
					return err
				}
				if user.ID != "fetched-1" {
					return fmt.Errorf("expected fetched user ID 'fetched-1', got '%s'", user.ID)
				}
				return nil
			},
			expectedCalls: []string{"Get"},
		},
		{
			name: "GetByID_CacheMiss",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.getByIDResult = TestUser{ID: "user-2", Name: "Fetched User 2"}
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user, err := cached.GetByID(context.Background(), "user-2")
				if err != nil {
					return err
				}
				if user.ID != "user-2" {
					return fmt.Errorf("expected user ID 'user-2', got '%s'", user.ID)
				}
				return nil
			},
			expectedCalls: []string{"GetByID"},
		},
		{
			name: "List_CacheMiss",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.listRecords = []TestUser{{ID: "3", Name: "User 3"}, {ID: "4", Name: "User 4"}}
				repo.listTotal = 2
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				records, total, err := cached.List(context.Background())
				if err != nil {
					return err
				}
				if len(records) != 2 || total != 2 {
					return fmt.Errorf("expected 2 records and total 2, got %d records and total %d", len(records), total)
				}
				if records[0].ID != "3" {
					return fmt.Errorf("expected first record ID '3', got '%s'", records[0].ID)
				}
				return nil
			},
			expectedCalls: []string{"List"},
		},
		{
			name: "Count_CacheMiss",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.countResult = 100
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				count, err := cached.Count(context.Background())
				if err != nil {
					return err
				}
				if count != 100 {
					return fmt.Errorf("expected count 100, got %d", count)
				}
				return nil
			},
			expectedCalls: []string{"Count"},
		},
		{
			name: "GetByIdentifier_CacheMiss",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.getByIDResult2 = TestUser{ID: "user-3", Name: "User by identifier from repo"}
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user, err := cached.GetByIdentifier(context.Background(), "email@example.com")
				if err != nil {
					return err
				}
				if user.Name != "User by identifier from repo" {
					return fmt.Errorf("expected user name 'User by identifier from repo', got '%s'", user.Name)
				}
				return nil
			},
			expectedCalls: []string{"GetByIdentifier"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseRepo := &mockRepository[TestUser]{}
			cacheService := newMockCacheService()
			keySerializer := newTrackingKeySerializer()

			tt.setupRepo(baseRepo)

			cached := New(baseRepo, cacheService, keySerializer)

			err := tt.testOperation(cached)
			if err != nil {
				t.Fatalf("test operation failed: %v", err)
			}

			// Verify base repository was called (cache miss)
			calls := baseRepo.getCalls()
			if len(calls) != len(tt.expectedCalls) {
				t.Errorf("expected %d base repo calls, got %d: %v", len(tt.expectedCalls), len(calls), calls)
			}
			for i, expected := range tt.expectedCalls {
				if i >= len(calls) || calls[i] != expected {
					t.Errorf("expected call %d to be '%s', got '%s'", i, expected, calls[i])
				}
			}
		})
	}
}

func TestCachedRepository_ScopeAwareKeys(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := cache.NewDefaultKeySerializer()

	cached := New(baseRepo, cacheService, keySerializer)

	ctxTenantA := repository.WithSelectScopes(context.Background(), "tenant")
	ctxTenantA = repository.WithScopeData(ctxTenantA, "tenant", "tenant-a")

	signatureA := cached.scopeSignature(ctxTenantA, repository.ScopeOperationSelect)
	if signatureA.IsZero() {
		t.Fatalf("expected scope signature for tenant A to be non-zero")
	}

	keyA := cached.key("Get", signatureA)

	baseRepo.getResult = TestUser{ID: "tenant-a", Name: "Tenant A"}
	userA, err := cached.Get(ctxTenantA)
	if err != nil {
		t.Fatalf("unexpected error fetching tenant A: %v", err)
	}
	if userA.ID != "tenant-a" {
		t.Fatalf("expected tenant-a record, got %s", userA.ID)
	}

	cacheService.mu.Lock()
	_, ok := cacheService.storage[keyA]
	cacheService.mu.Unlock()
	if !ok {
		t.Fatalf("expected cache entry for tenant A key %s", keyA)
	}

	ctxTenantB := repository.WithSelectScopes(context.Background(), "tenant")
	ctxTenantB = repository.WithScopeData(ctxTenantB, "tenant", "tenant-b")
	signatureB := cached.scopeSignature(ctxTenantB, repository.ScopeOperationSelect)
	if signatureB.IsZero() {
		t.Fatalf("expected scope signature for tenant B to be non-zero")
	}

	keyB := cached.key("Get", signatureB)
	if keyA == keyB {
		t.Fatalf("expected different cache keys for different tenant scopes, got %s", keyA)
	}

	baseRepo.getResult = TestUser{ID: "tenant-b", Name: "Tenant B"}
	userB, err := cached.Get(ctxTenantB)
	if err != nil {
		t.Fatalf("unexpected error fetching tenant B: %v", err)
	}
	if userB.ID != "tenant-b" {
		t.Fatalf("expected tenant-b record, got %s", userB.ID)
	}

	cacheService.mu.Lock()
	_, ok = cacheService.storage[keyB]
	cacheService.mu.Unlock()
	if !ok {
		t.Fatalf("expected cache entry for tenant B key %s", keyB)
	}

	calls := baseRepo.getCalls()
	if len(calls) != 2 {
		t.Fatalf("expected base repository Get to be called twice, got %d calls: %v", len(calls), calls)
	}

	baseRepo.clearCalls()

	// Subsequent access for tenant A should hit cache
	_, err = cached.Get(ctxTenantA)
	if err != nil {
		t.Fatalf("unexpected error retrieving cached tenant A: %v", err)
	}

	if calls := baseRepo.getCalls(); len(calls) != 0 {
		t.Fatalf("expected no additional base repo calls, got %v", calls)
	}
}

// Test error propagation for read methods
func TestCachedReadMethods_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name          string
		setupCache    func(*mockCacheService, *CachedRepository[TestUser])
		setupRepo     func(*mockRepository[TestUser])
		testOperation func(*CachedRepository[TestUser]) error
		expectedError string
	}{
		{
			name:       "Get_RepoError",
			setupCache: func(cache *mockCacheService, _ *CachedRepository[TestUser]) {},
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.getError = errors.New("repository error")
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				_, err := cached.Get(context.Background())
				return err
			},
			expectedError: "repository error",
		},
		{
			name: "Get_CacheError",
			setupCache: func(cache *mockCacheService, cached *CachedRepository[TestUser]) {
				cache.SetCacheError(cached.key("Get"), errors.New("cache error"))
			},
			setupRepo: func(repo *mockRepository[TestUser]) {},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				_, err := cached.Get(context.Background())
				return err
			},
			expectedError: "cache error",
		},
		{
			name:       "List_RepoError",
			setupCache: func(cache *mockCacheService, _ *CachedRepository[TestUser]) {},
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.listError = errors.New("list repository error")
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				_, _, err := cached.List(context.Background())
				return err
			},
			expectedError: "list repository error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseRepo := &mockRepository[TestUser]{}
			cacheService := newMockCacheService()
			keySerializer := newTrackingKeySerializer()

			tt.setupRepo(baseRepo)

			cached := New(baseRepo, cacheService, keySerializer)
			tt.setupCache(cacheService, cached)

			err := tt.testOperation(cached)
			if err == nil {
				t.Fatalf("expected error '%s', got nil", tt.expectedError)
			}
			if err.Error() != tt.expectedError {
				t.Errorf("expected error '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

// Test write method delegation
func TestWriteMethodsDelegation(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func(*mockRepository[TestUser])
		testOperation func(*CachedRepository[TestUser]) error
		expectedCalls []string
		expectedError string
	}{
		{
			name: "Create_Success",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.createResult = TestUser{ID: "new-user", Name: "New User"}
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user := TestUser{Name: "New User"}
				result, err := cached.Create(context.Background(), user)
				if err != nil {
					return err
				}
				if result.ID != "new-user" {
					return fmt.Errorf("expected created user ID 'new-user', got '%s'", result.ID)
				}
				return nil
			},
			expectedCalls: []string{"Create"},
		},
		{
			name: "Create_Error",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.createError = errors.New("create failed")
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user := TestUser{Name: "New User"}
				_, err := cached.Create(context.Background(), user)
				return err
			},
			expectedCalls: []string{"Create"},
			expectedError: "create failed",
		},
		{
			name: "Update_Success",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.updateResult = TestUser{ID: "updated-user", Name: "Updated User"}
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user := TestUser{ID: "updated-user", Name: "Updated User"}
				result, err := cached.Update(context.Background(), user)
				if err != nil {
					return err
				}
				if result.Name != "Updated User" {
					return fmt.Errorf("expected updated user name 'Updated User', got '%s'", result.Name)
				}
				return nil
			},
			expectedCalls: []string{"Update"},
		},
		{
			name: "Delete_Success",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.deleteError = nil
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user := TestUser{ID: "delete-user", Name: "Delete User"}
				return cached.Delete(context.Background(), user)
			},
			expectedCalls: []string{"Delete"},
		},
		{
			name: "Delete_Error",
			setupRepo: func(repo *mockRepository[TestUser]) {
				repo.deleteError = errors.New("delete failed")
			},
			testOperation: func(cached *CachedRepository[TestUser]) error {
				user := TestUser{ID: "delete-user", Name: "Delete User"}
				return cached.Delete(context.Background(), user)
			},
			expectedCalls: []string{"Delete"},
			expectedError: "delete failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseRepo := &mockRepository[TestUser]{}
			cacheService := newMockCacheService()
			keySerializer := newTrackingKeySerializer()

			tt.setupRepo(baseRepo)

			cached := New(baseRepo, cacheService, keySerializer)

			err := tt.testOperation(cached)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error '%s', got nil", tt.expectedError)
				}
				if err.Error() != tt.expectedError {
					t.Errorf("expected error '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify base repository was called
			calls := baseRepo.getCalls()
			if len(calls) != len(tt.expectedCalls) {
				t.Errorf("expected %d base repo calls, got %d: %v", len(tt.expectedCalls), len(calls), calls)
			}
			for i, expected := range tt.expectedCalls {
				if i >= len(calls) || calls[i] != expected {
					t.Errorf("expected call %d to be '%s', got '%s'", i, expected, calls[i])
				}
			}
		})
	}
}

// Test key serializer integration
func TestKeySerializerIntegration(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	// Configure repo to return data
	baseRepo.getResult = TestUser{ID: "user-1", Name: "Test User"}
	baseRepo.getByIDResult = TestUser{ID: "user-2", Name: "User by ID"}
	baseRepo.listRecords = []TestUser{{ID: "user-3", Name: "User 3"}}
	baseRepo.listTotal = 1
	baseRepo.countResult = 5
	baseRepo.getByIDResult2 = TestUser{ID: "user-4", Name: "User by identifier"}

	cached := New(baseRepo, cacheService, keySerializer)

	// Test various operations to verify key serializer is called with correct parameters
	_, _ = cached.Get(context.Background())
	_, _ = cached.GetByID(context.Background(), "test-id")
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())
	_, _ = cached.GetByIdentifier(context.Background(), "test-identifier")

	// Verify key serializer was called
	calls := keySerializer.getCalls()
	expectedCalls := []string{
		cached.methodKey("Get") + ":[]",
		cached.methodKey("GetByID") + ":[test-id]",
		cached.methodKey("List") + ":[]",
		cached.methodKey("Count") + ":[]",
		cached.methodKey("GetByIdentifier") + ":[test-identifier]",
	}

	if len(calls) != len(expectedCalls) {
		t.Errorf("expected %d key serializer calls, got %d: %v", len(expectedCalls), len(calls), calls)
	}

	for i, expected := range expectedCalls {
		if i >= len(calls) || calls[i] != expected {
			t.Errorf("expected key serializer call %d to be '%s', got '%s'", i, expected, calls[i])
		}
	}
}

// Test repository interface satisfaction
func TestRepositoryInterfaceSatisfaction(t *testing.T) {
	// This test ensures our CachedRepository satisfies the Repository interface
	// The interface assertion at the bottom of decorator.go should catch this at compile time,
	// but this test provides runtime verification as well.

	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	cached := New(baseRepo, cacheService, keySerializer)

	// Verify that our cached repository can be assigned to the interface
	var repo repository.Repository[TestUser] = cached
	if repo == nil {
		t.Error("CachedRepository does not satisfy Repository interface")
	}
}

// Test fixture-based scenarios using test support utilities
func TestCacheScenarios_WithFixtures(t *testing.T) {
	// This test demonstrates how fixtures could be used for more complex scenarios
	// For now, it's a simple example showing the pattern

	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	// Example fixture data (normally would be loaded from testdata/)
	testUsers := []TestUser{
		{ID: "fixture-1", Name: "Fixture User 1"},
		{ID: "fixture-2", Name: "Fixture User 2"},
	}

	// Configure repository mock with fixture data
	baseRepo.listRecords = testUsers
	baseRepo.listTotal = len(testUsers)

	cached := New(baseRepo, cacheService, keySerializer)

	// First call should hit repository (cache miss)
	records1, total1, err := cached.List(context.Background())
	if err != nil {
		t.Fatalf("First List call failed: %v", err)
	}

	if len(records1) != 2 || total1 != 2 {
		t.Errorf("Expected 2 records and total 2 from first call, got %d records and total %d", len(records1), total1)
	}

	// Clear repository call tracking
	baseRepo.clearCalls()

	// Second call should hit cache (cache hit)
	records2, total2, err := cached.List(context.Background())
	if err != nil {
		t.Fatalf("Second List call failed: %v", err)
	}

	if len(records2) != 2 || total2 != 2 {
		t.Errorf("Expected 2 records and total 2 from second call, got %d records and total %d", len(records2), total2)
	}

	// Verify repository was not called on second invocation (cache hit)
	calls := baseRepo.getCalls()
	if len(calls) != 0 {
		t.Errorf("Expected no repository calls on cache hit, got %d: %v", len(calls), calls)
	}

	// Verify the results are the same
	if !reflect.DeepEqual(records1, records2) {
		t.Error("Results from cache hit should match results from cache miss")
	}
}

// ---- Invalidation Tests ----

// Test cache invalidation after create operations
func TestCacheInvalidation_Create(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	// Set up initial repository state
	baseRepo.listRecords = []TestUser{
		{ID: "user-1", Name: "User 1"},
		{ID: "user-2", Name: "User 2"},
	}
	baseRepo.listTotal = 2
	baseRepo.countResult = 2

	cached := New(baseRepo, cacheService, keySerializer)

	// Populate cache with List and Count operations
	records1, total1, err := cached.List(context.Background())
	if err != nil {
		t.Fatalf("Initial List call failed: %v", err)
	}
	if len(records1) != 2 || total1 != 2 {
		t.Fatalf("Expected 2 records and total 2, got %d records and total %d", len(records1), total1)
	}

	count1, err := cached.Count(context.Background())
	if err != nil {
		t.Fatalf("Initial Count call failed: %v", err)
	}
	if count1 != 2 {
		t.Fatalf("Expected count 2, got %d", count1)
	}

	// Clear call tracking to verify cache hits
	baseRepo.clearCalls()

	// Verify cache hits (should not call repository)
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())
	calls := baseRepo.getCalls()
	if len(calls) != 0 {
		t.Fatalf("Expected no repository calls (cache hit), got %d: %v", len(calls), calls)
	}

	// Now create a new record
	baseRepo.createResult = TestUser{ID: "user-3", Name: "User 3"}
	// Update repository state to reflect the new record
	baseRepo.listRecords = []TestUser{
		{ID: "user-1", Name: "User 1"},
		{ID: "user-2", Name: "User 2"},
		{ID: "user-3", Name: "User 3"},
	}
	baseRepo.listTotal = 3
	baseRepo.countResult = 3

	newUser := TestUser{Name: "User 3"}
	_, err = cached.Create(context.Background(), newUser)
	if err != nil {
		t.Fatalf("Create operation failed: %v", err)
	}

	// Clear call tracking
	baseRepo.clearCalls()

	// Verify that List and Count caches were invalidated (should call repository)
	records2, total2, err := cached.List(context.Background())
	if err != nil {
		t.Fatalf("List after create failed: %v", err)
	}
	if len(records2) != 3 || total2 != 3 {
		t.Errorf("Expected 3 records and total 3 after create, got %d records and total %d", len(records2), total2)
	}

	count2, err := cached.Count(context.Background())
	if err != nil {
		t.Fatalf("Count after create failed: %v", err)
	}
	if count2 != 3 {
		t.Errorf("Expected count 3 after create, got %d", count2)
	}

	// Verify repository was called (cache invalidated)
	calls = baseRepo.getCalls()
	expectedCalls := []string{"List", "Count"}
	if len(calls) != len(expectedCalls) {
		t.Errorf("Expected %d repository calls after cache invalidation, got %d: %v", len(expectedCalls), len(calls), calls)
	}

	// Verify cache invalidation was called
	cacheCalls := cacheService.getCalls()
	found := false
	for _, call := range cacheCalls {
		if strings.Contains(call, cached.methodKey("List")) || strings.Contains(call, cached.methodKey("Count")) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected cache Delete calls for List and Count prefixes after create operation")
	}
}

// Test cache invalidation after update operations
func TestCacheInvalidation_Update(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	originalUser := TestUser{ID: "user-1", Name: "Original User"}
	updatedUser := TestUser{ID: "user-1", Name: "Updated User"}

	// Set up repository to return original user initially
	baseRepo.getByIDResult = originalUser
	baseRepo.listRecords = []TestUser{originalUser}
	baseRepo.listTotal = 1
	baseRepo.countResult = 1

	cached := New(baseRepo, cacheService, keySerializer)

	// Populate cache with GetByID, List, and Count operations
	user1, err := cached.GetByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Initial GetByID call failed: %v", err)
	}
	if user1.Name != "Original User" {
		t.Fatalf("Expected 'Original User', got '%s'", user1.Name)
	}

	_, _, err = cached.List(context.Background())
	if err != nil {
		t.Fatalf("Initial List call failed: %v", err)
	}

	_, err = cached.Count(context.Background())
	if err != nil {
		t.Fatalf("Initial Count call failed: %v", err)
	}

	// Clear call tracking to verify cache hits
	baseRepo.clearCalls()

	// Verify cache hits
	_, _ = cached.GetByID(context.Background(), "user-1")
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())
	calls := baseRepo.getCalls()
	if len(calls) != 0 {
		t.Fatalf("Expected no repository calls (cache hit), got %d: %v", len(calls), calls)
	}

	// Now update the user
	baseRepo.updateResult = updatedUser
	// Update repository state to reflect the updated record
	baseRepo.getByIDResult = updatedUser
	baseRepo.listRecords = []TestUser{updatedUser}

	_, err = cached.Update(context.Background(), updatedUser)
	if err != nil {
		t.Fatalf("Update operation failed: %v", err)
	}

	// Clear call tracking
	baseRepo.clearCalls()

	// Verify that relevant caches were invalidated
	user2, err := cached.GetByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if user2.Name != "Updated User" {
		t.Errorf("Expected 'Updated User' after update, got '%s'", user2.Name)
	}

	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())

	// Verify repository was called (cache invalidated)
	calls = baseRepo.getCalls()
	if len(calls) < 3 {
		t.Errorf("Expected at least 3 repository calls after cache invalidation, got %d: %v", len(calls), calls)
	}

	// Verify cache invalidation was called for relevant prefixes
	cacheCalls := cacheService.getCalls()
	expectedSubstrings := []string{
		cached.methodPrefix("GetByID", "user-1"),
		cached.methodKey("List"),
		cached.methodKey("Count"),
	}
	for _, expected := range expectedSubstrings {
		found := false
		for _, call := range cacheCalls {
			if strings.Contains(call, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected cache invalidation containing '%s' after update operation, got calls: %v", expected, cacheCalls)
		}
	}
}

func TestCacheInvalidation_UpdateWithScopedGetByID(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := cache.NewDefaultKeySerializer()

	originalUser := TestUser{ID: "user-1", Name: "Original User"}
	updatedUser := TestUser{ID: "user-1", Name: "Updated User"}

	baseRepo.getByIDResult = originalUser

	cached := New(baseRepo, cacheService, keySerializer)

	ctx := repository.WithSelectScopes(context.Background(), "tenant")
	ctx = repository.WithScopeData(ctx, "tenant", "tenant-a")

	signature := cached.scopeSignature(ctx, repository.ScopeOperationSelect)
	if signature.IsZero() {
		t.Fatalf("Expected non-zero scope signature")
	}

	key := cached.key("GetByID", "user-1", signature)

	user, err := cached.GetByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("Initial GetByID call failed: %v", err)
	}
	if user.Name != "Original User" {
		t.Fatalf("Expected 'Original User', got '%s'", user.Name)
	}

	cacheService.mu.Lock()
	_, exists := cacheService.storage[key]
	cacheService.mu.Unlock()
	if !exists {
		t.Fatalf("Expected cache entry for scoped GetByID key %s", key)
	}

	baseRepo.updateResult = updatedUser
	baseRepo.getByIDResult = updatedUser

	if _, err := cached.Update(context.Background(), updatedUser); err != nil {
		t.Fatalf("Update operation failed: %v", err)
	}

	cacheService.mu.Lock()
	_, exists = cacheService.storage[key]
	cacheService.mu.Unlock()
	if exists {
		t.Fatalf("Expected scoped GetByID cache entry %s to be invalidated after update", key)
	}

	baseRepo.clearCalls()
	baseRepo.getByIDResult = updatedUser

	result, err := cached.GetByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if result.Name != "Updated User" {
		t.Fatalf("Expected 'Updated User' after update, got '%s'", result.Name)
	}

	calls := baseRepo.getCalls()
	if len(calls) != 1 || calls[0] != "GetByID" {
		t.Fatalf("Expected base repository GetByID to be called once after invalidation, got %v", calls)
	}

	deletePrefix := cached.methodPrefix("GetByID", "user-1")
	cacheCalls := cacheService.getCalls()
	foundPrefix := false
	for _, call := range cacheCalls {
		if strings.Contains(call, deletePrefix) {
			foundPrefix = true
			break
		}
	}
	if !foundPrefix {
		t.Errorf("Expected cache invalidation call containing '%s', got calls: %v", deletePrefix, cacheCalls)
	}
}

// Test cache invalidation after delete operations
func TestCacheInvalidation_Delete(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	userToDelete := TestUser{ID: "user-1", Name: "User to Delete"}

	// Set up repository state
	baseRepo.getByIDResult = userToDelete
	baseRepo.listRecords = []TestUser{userToDelete, {ID: "user-2", Name: "User 2"}}
	baseRepo.listTotal = 2
	baseRepo.countResult = 2

	cached := New(baseRepo, cacheService, keySerializer)

	// Populate cache
	_, _ = cached.GetByID(context.Background(), "user-1")
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())

	// Clear call tracking to verify cache hits
	baseRepo.clearCalls()

	// Verify cache hits
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())
	calls := baseRepo.getCalls()
	if len(calls) != 0 {
		t.Fatalf("Expected no repository calls (cache hit), got %d: %v", len(calls), calls)
	}

	// Now delete the user
	// Update repository state to reflect deletion
	baseRepo.listRecords = []TestUser{{ID: "user-2", Name: "User 2"}}
	baseRepo.listTotal = 1
	baseRepo.countResult = 1

	err := cached.Delete(context.Background(), userToDelete)
	if err != nil {
		t.Fatalf("Delete operation failed: %v", err)
	}

	// Clear call tracking
	baseRepo.clearCalls()

	// Verify that relevant caches were invalidated
	records, total, err := cached.List(context.Background())
	if err != nil {
		t.Fatalf("List after delete failed: %v", err)
	}
	if len(records) != 1 || total != 1 {
		t.Errorf("Expected 1 record and total 1 after delete, got %d records and total %d", len(records), total)
	}

	count, err := cached.Count(context.Background())
	if err != nil {
		t.Fatalf("Count after delete failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 after delete, got %d", count)
	}

	// Verify repository was called (cache invalidated)
	calls = baseRepo.getCalls()
	expectedCalls := []string{"List", "Count"}
	if len(calls) < len(expectedCalls) {
		t.Errorf("Expected at least %d repository calls after cache invalidation, got %d: %v", len(expectedCalls), len(calls), calls)
	}

	// Verify cache invalidation was called for relevant prefixes
	cacheCalls := cacheService.getCalls()
	expectedDeleteSubstrings := []string{
		cached.methodPrefix("GetByID", "user-1"),
		cached.methodKey("List"),
		cached.methodKey("Count"),
	}
	for _, expected := range expectedDeleteSubstrings {
		found := false
		for _, call := range cacheCalls {
			if strings.Contains(call, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected cache invalidation containing '%s' after delete operation, got calls: %v", expected, cacheCalls)
		}
	}
}

// Test bulk operations cache invalidation
func TestCacheInvalidation_BulkOperations(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	// Set up initial state
	baseRepo.listRecords = []TestUser{{ID: "user-1", Name: "User 1"}}
	baseRepo.listTotal = 1
	baseRepo.countResult = 1

	cached := New(baseRepo, cacheService, keySerializer)

	// Populate cache
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())

	// Clear call tracking
	baseRepo.clearCalls()

	// Verify cache hits
	_, _, _ = cached.List(context.Background())
	calls := baseRepo.getCalls()
	if len(calls) != 0 {
		t.Fatalf("Expected no repository calls (cache hit), got %d: %v", len(calls), calls)
	}

	// Test CreateMany
	newUsers := []TestUser{
		{Name: "User 2"},
		{Name: "User 3"},
	}
	baseRepo.createError = nil // Ensure no error
	_, err := cached.CreateMany(context.Background(), newUsers)
	if err != nil {
		t.Fatalf("CreateMany operation failed: %v", err)
	}

	// Update repository state
	baseRepo.listRecords = []TestUser{
		{ID: "user-1", Name: "User 1"},
		{ID: "user-2", Name: "User 2"},
		{ID: "user-3", Name: "User 3"},
	}
	baseRepo.listTotal = 3
	baseRepo.countResult = 3

	// Clear call tracking
	baseRepo.clearCalls()

	// Verify cache was invalidated (should call repository)
	_, _, err = cached.List(context.Background())
	if err != nil {
		t.Fatalf("List after CreateMany failed: %v", err)
	}

	// Verify repository was called (cache invalidated)
	calls = baseRepo.getCalls()
	if len(calls) == 0 {
		t.Error("Expected repository calls after CreateMany cache invalidation")
	}

	// Verify cache invalidation was called
	cacheCalls := cacheService.getCalls()
	found := false
	for _, call := range cacheCalls {
		if strings.Contains(call, cached.methodKey("List")) || strings.Contains(call, cached.methodKey("Count")) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected cache Delete calls after CreateMany operation")
	}
}

// Test criteria-based operations cache invalidation
func TestCacheInvalidation_CriteriaOperations(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	// Set up initial state
	baseRepo.getByIDResult = TestUser{ID: "user-1", Name: "User 1"}
	baseRepo.listRecords = []TestUser{{ID: "user-1", Name: "User 1"}}
	baseRepo.listTotal = 1
	baseRepo.countResult = 1

	cached := New(baseRepo, cacheService, keySerializer)

	// Populate cache
	_, _ = cached.GetByID(context.Background(), "user-1")
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())

	// Clear call tracking
	baseRepo.clearCalls()

	// Test DeleteMany (criteria-based operation)
	err := cached.DeleteMany(context.Background())
	if err != nil {
		t.Fatalf("DeleteMany operation failed: %v", err)
	}

	// Update repository state
	baseRepo.listRecords = []TestUser{}
	baseRepo.listTotal = 0
	baseRepo.countResult = 0
	// The user should no longer exist after DeleteMany
	baseRepo.getByIDError = fmt.Errorf("user not found")

	// Clear call tracking
	baseRepo.clearCalls()

	// Verify all relevant caches were invalidated
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())

	// Verify repository was called (cache invalidated)
	calls := baseRepo.getCalls()
	if len(calls) < 2 {
		t.Error("Expected repository calls after DeleteMany cache invalidation")
	}

	// Verify comprehensive cache invalidation was called
	cacheCalls := cacheService.getCalls()
	// For criteria operations, we should see Delete calls for prefixes that actually have cached keys
	// In our test, we only cached GetByID, List, and Count, so only those should have Delete calls
	expectedDeleteSubstrings := []string{
		cached.methodPrefixWithSeparator("GetByID"),
		cached.methodKey("List"),
		cached.methodKey("Count"),
	}
	for _, substr := range expectedDeleteSubstrings {
		found := false
		for _, call := range cacheCalls {
			if strings.Contains(call, substr) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected cache invalidation containing '%s' after DeleteMany operation, got calls: %v", substr, cacheCalls)
		}
	}

	// Verify that invalidateAfterCriteriaOperation was called by checking the behavior:
	// After DeleteMany, if we try to access the same data, it should call the repository (cache miss)
	calls = baseRepo.getCalls()
	initialCallCount := len(calls)

	// Try accessing data that should have been invalidated
	_, _ = cached.GetByID(context.Background(), "user-1")
	calls = baseRepo.getCalls()
	if len(calls) <= initialCallCount {
		t.Error("Expected additional repository call after cache invalidation for GetByID")
	}
}

// Test negative caching behavior when records are deleted
func TestCacheInvalidation_NegativeCaching(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	userToDelete := TestUser{ID: "user-1", Name: "User to Delete"}

	// Set up repository to return user initially
	baseRepo.getByIDResult = userToDelete

	cached := New(baseRepo, cacheService, keySerializer)

	// First, get the user to populate cache
	user1, err := cached.GetByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Initial GetByID failed: %v", err)
	}
	if user1.Name != "User to Delete" {
		t.Fatalf("Expected 'User to Delete', got '%s'", user1.Name)
	}

	// Clear call tracking
	baseRepo.clearCalls()

	// Verify cache hit
	_, _ = cached.GetByID(context.Background(), "user-1")
	calls := baseRepo.getCalls()
	if len(calls) != 0 {
		t.Fatalf("Expected no repository calls (cache hit), got %d: %v", len(calls), calls)
	}

	// Now delete the user
	err = cached.Delete(context.Background(), userToDelete)
	if err != nil {
		t.Fatalf("Delete operation failed: %v", err)
	}

	// Update repository to return error when trying to get deleted user
	baseRepo.getByIDError = fmt.Errorf("user not found")

	// Clear call tracking
	baseRepo.clearCalls()

	// Try to get the deleted user - should call repository and get error
	_, err = cached.GetByID(context.Background(), "user-1")
	if err == nil {
		t.Error("Expected error when trying to get deleted user")
	}
	if err.Error() != "user not found" {
		t.Errorf("Expected 'user not found' error, got '%v'", err)
	}

	// Verify repository was called (cache was invalidated)
	calls = baseRepo.getCalls()
	if len(calls) == 0 {
		t.Error("Expected repository call after cache invalidation for deleted user")
	}
}

// Test concurrent cache invalidation
func TestCacheInvalidation_Concurrent(t *testing.T) {
	baseRepo := &mockRepository[TestUser]{}
	cacheService := newMockCacheService()
	keySerializer := newTrackingKeySerializer()

	// Set up initial state
	baseRepo.listRecords = []TestUser{{ID: "user-1", Name: "User 1"}}
	baseRepo.listTotal = 1
	baseRepo.countResult = 1
	baseRepo.createResult = TestUser{ID: "user-2", Name: "User 2"}

	cached := New(baseRepo, cacheService, keySerializer)

	// Populate cache
	_, _, _ = cached.List(context.Background())
	_, _ = cached.Count(context.Background())

	// Test concurrent create operations that should trigger invalidation
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			newUser := TestUser{Name: fmt.Sprintf("User %d", id+10)}
			_, err := cached.Create(context.Background(), newUser)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent create failed: %v", err)
	}

	// Verify that cache invalidation was called multiple times
	cacheCalls := cacheService.getCalls()
	deleteCount := 0
	for _, call := range cacheCalls {
		if strings.Contains(call, cached.methodKey("List")) || strings.Contains(call, cached.methodKey("Count")) {
			deleteCount++
		}
	}

	if deleteCount == 0 {
		t.Error("Expected cache Delete calls from concurrent operations")
	}
}
