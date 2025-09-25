package repositorycache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	repository "github.com/goliatone/go-repository-bun"
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
	panic("CreateMany not implemented in mock")
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
	panic("DeleteMany not implemented in mock")
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

	// Default key generation
	return fmt.Sprintf("%s_%s", method, argStr)
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

	if cached.cache != cacheService {
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
		setupCache    func(*mockCacheService)
		setupRepo     func(*mockRepository[TestUser])
		testOperation func(*CachedRepository[TestUser]) error
		expectedCalls []string
	}{
		{
			name: "Get_CacheHit",
			setupCache: func(cache *mockCacheService) {
				cache.SetCacheValue("Get_[[]]", TestUser{ID: "cached-1", Name: "Cached User"})
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
			setupCache: func(cache *mockCacheService) {
				cache.SetCacheValue("GetByID_[user-1 []]", TestUser{ID: "user-1", Name: "Cached User"})
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
			setupCache: func(cache *mockCacheService) {
				result := listResult[TestUser]{
					Records: []TestUser{{ID: "1", Name: "User 1"}, {ID: "2", Name: "User 2"}},
					Total:   2,
				}
				cache.SetCacheValue("List_[[]]", result)
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
			setupCache: func(cache *mockCacheService) {
				cache.SetCacheValue("Count_[[]]", 42)
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
			setupCache: func(cache *mockCacheService) {
				cache.SetCacheValue("GetByIdentifier_[username123 []]", TestUser{ID: "user-1", Name: "User by identifier"})
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

			tt.setupCache(cacheService)
			tt.setupRepo(baseRepo)

			cached := New(baseRepo, cacheService, keySerializer)

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

// Test error propagation for read methods
func TestCachedReadMethods_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name          string
		setupCache    func(*mockCacheService)
		setupRepo     func(*mockRepository[TestUser])
		testOperation func(*CachedRepository[TestUser]) error
		expectedError string
	}{
		{
			name:       "Get_RepoError",
			setupCache: func(cache *mockCacheService) {},
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
			setupCache: func(cache *mockCacheService) {
				cache.SetCacheError("Get_[[]]", errors.New("cache error"))
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
			setupCache: func(cache *mockCacheService) {},
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

			tt.setupCache(cacheService)
			tt.setupRepo(baseRepo)

			cached := New(baseRepo, cacheService, keySerializer)

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
		"Get:[[]]",
		"GetByID:[test-id []]",
		"List:[[]]",
		"Count:[[]]",
		"GetByIdentifier:[test-identifier []]",
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
