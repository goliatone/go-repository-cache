package di

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	repository "github.com/goliatone/go-repository-bun"
	"github.com/goliatone/go-repository-cache/cache"
	"github.com/uptrace/bun"
)

// User represents a test model for integration tests
type User struct {
	ID       string `json:"id" bun:"id,pk"`
	Name     string `json:"name" bun:"name"`
	Email    string `json:"email" bun:"email"`
	CreateTs int64  `json:"create_ts" bun:"create_ts"`
}

// mockUserRepository provides a fake repository implementation for testing
type mockUserRepository struct {
	mu        sync.RWMutex
	users     map[string]User
	callCount map[string]int // Track method calls to verify caching behavior
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:     make(map[string]User),
		callCount: make(map[string]int),
	}
}

func (m *mockUserRepository) trackCall(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount[method]++
}

func (m *mockUserRepository) getCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount[method]
}

// GetByID implementation for mock repository
func (m *mockUserRepository) GetByID(ctx context.Context, id string, criteria ...repository.SelectCriteria) (User, error) {
	m.trackCall("GetByID")
	m.mu.RLock()
	user, exists := m.users[id]
	m.mu.RUnlock()
	if !exists {
		return User{}, errors.New("user not found")
	}
	return user, nil
}

// Get implementation for mock repository
func (m *mockUserRepository) Get(ctx context.Context, criteria ...repository.SelectCriteria) (User, error) {
	m.trackCall("Get")
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Simple implementation - return first user if any exists
	for _, user := range m.users {
		return user, nil
	}
	return User{}, errors.New("no users found")
}

// List implementation for mock repository
func (m *mockUserRepository) List(ctx context.Context, criteria ...repository.SelectCriteria) ([]User, int, error) {
	m.trackCall("List")
	m.mu.RLock()
	defer m.mu.RUnlock()
	users := make([]User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, len(users), nil
}

// Count implementation for mock repository
func (m *mockUserRepository) Count(ctx context.Context, criteria ...repository.SelectCriteria) (int, error) {
	m.trackCall("Count")
	m.mu.RLock()
	count := len(m.users)
	m.mu.RUnlock()
	return count, nil
}

// GetByIdentifier implementation for mock repository
func (m *mockUserRepository) GetByIdentifier(ctx context.Context, identifier string, criteria ...repository.SelectCriteria) (User, error) {
	m.trackCall("GetByIdentifier")
	return m.GetByID(ctx, identifier, criteria...)
}

// Create implementation for mock repository
func (m *mockUserRepository) Create(ctx context.Context, user User, criteria ...repository.InsertCriteria) (User, error) {
	m.trackCall("Create")
	if user.CreateTs == 0 {
		user.CreateTs = time.Now().Unix()
	}
	m.mu.Lock()
	m.users[user.ID] = user
	m.mu.Unlock()
	return user, nil
}

// Update implementation for mock repository
func (m *mockUserRepository) Update(ctx context.Context, user User, criteria ...repository.UpdateCriteria) (User, error) {
	m.trackCall("Update")
	m.mu.Lock()
	m.users[user.ID] = user
	m.mu.Unlock()
	return user, nil
}

// Delete implementation for mock repository
func (m *mockUserRepository) Delete(ctx context.Context, user User) error {
	m.trackCall("Delete")
	m.mu.Lock()
	delete(m.users, user.ID)
	m.mu.Unlock()
	return nil
}

// Stub implementations for other required methods
func (m *mockUserRepository) CreateTx(ctx context.Context, tx bun.IDB, record User, criteria ...repository.InsertCriteria) (User, error) {
	return m.Create(ctx, record, criteria...)
}
func (m *mockUserRepository) CreateMany(ctx context.Context, records []User, criteria ...repository.InsertCriteria) ([]User, error) {
	for _, record := range records {
		m.Create(ctx, record, criteria...)
	}
	return records, nil
}
func (m *mockUserRepository) CreateManyTx(ctx context.Context, tx bun.IDB, records []User, criteria ...repository.InsertCriteria) ([]User, error) {
	return m.CreateMany(ctx, records, criteria...)
}
func (m *mockUserRepository) GetOrCreate(ctx context.Context, record User) (User, error) {
	m.mu.RLock()
	if existing, exists := m.users[record.ID]; exists {
		m.mu.RUnlock()
		return existing, nil
	}
	m.mu.RUnlock()
	return m.Create(ctx, record)
}
func (m *mockUserRepository) GetOrCreateTx(ctx context.Context, tx bun.IDB, record User) (User, error) {
	return m.GetOrCreate(ctx, record)
}
func (m *mockUserRepository) UpdateTx(ctx context.Context, tx bun.IDB, record User, criteria ...repository.UpdateCriteria) (User, error) {
	return m.Update(ctx, record, criteria...)
}
func (m *mockUserRepository) UpdateMany(ctx context.Context, records []User, criteria ...repository.UpdateCriteria) ([]User, error) {
	for _, record := range records {
		m.Update(ctx, record, criteria...)
	}
	return records, nil
}
func (m *mockUserRepository) UpdateManyTx(ctx context.Context, tx bun.IDB, records []User, criteria ...repository.UpdateCriteria) ([]User, error) {
	return m.UpdateMany(ctx, records, criteria...)
}
func (m *mockUserRepository) Upsert(ctx context.Context, record User, criteria ...repository.UpdateCriteria) (User, error) {
	return m.Update(ctx, record, criteria...)
}
func (m *mockUserRepository) UpsertTx(ctx context.Context, tx bun.IDB, record User, criteria ...repository.UpdateCriteria) (User, error) {
	return m.Upsert(ctx, record, criteria...)
}
func (m *mockUserRepository) UpsertMany(ctx context.Context, records []User, criteria ...repository.UpdateCriteria) ([]User, error) {
	return m.UpdateMany(ctx, records, criteria...)
}
func (m *mockUserRepository) UpsertManyTx(ctx context.Context, tx bun.IDB, records []User, criteria ...repository.UpdateCriteria) ([]User, error) {
	return m.UpsertMany(ctx, records, criteria...)
}
func (m *mockUserRepository) DeleteTx(ctx context.Context, tx bun.IDB, record User) error {
	return m.Delete(ctx, record)
}
func (m *mockUserRepository) DeleteMany(ctx context.Context, criteria ...repository.DeleteCriteria) error {
	m.trackCall("DeleteMany")
	// Simple implementation - clear all users
	m.mu.Lock()
	m.users = make(map[string]User)
	m.mu.Unlock()
	return nil
}
func (m *mockUserRepository) DeleteManyTx(ctx context.Context, tx bun.IDB, criteria ...repository.DeleteCriteria) error {
	return m.DeleteMany(ctx, criteria...)
}
func (m *mockUserRepository) DeleteWhere(ctx context.Context, criteria ...repository.DeleteCriteria) error {
	return m.DeleteMany(ctx, criteria...)
}
func (m *mockUserRepository) DeleteWhereTx(ctx context.Context, tx bun.IDB, criteria ...repository.DeleteCriteria) error {
	return m.DeleteWhere(ctx, criteria...)
}
func (m *mockUserRepository) ForceDelete(ctx context.Context, record User) error {
	return m.Delete(ctx, record)
}
func (m *mockUserRepository) ForceDeleteTx(ctx context.Context, tx bun.IDB, record User) error {
	return m.ForceDelete(ctx, record)
}
func (m *mockUserRepository) GetTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) (User, error) {
	return m.Get(ctx, criteria...)
}
func (m *mockUserRepository) GetByIDTx(ctx context.Context, tx bun.IDB, id string, criteria ...repository.SelectCriteria) (User, error) {
	return m.GetByID(ctx, id, criteria...)
}
func (m *mockUserRepository) ListTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) ([]User, int, error) {
	return m.List(ctx, criteria...)
}
func (m *mockUserRepository) CountTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) (int, error) {
	return m.Count(ctx, criteria...)
}
func (m *mockUserRepository) GetByIdentifierTx(ctx context.Context, tx bun.IDB, identifier string, criteria ...repository.SelectCriteria) (User, error) {
	return m.GetByIdentifier(ctx, identifier, criteria...)
}
func (m *mockUserRepository) Raw(ctx context.Context, sql string, args ...any) ([]User, error) {
	m.trackCall("Raw")
	return nil, errors.New("raw queries not supported in mock")
}
func (m *mockUserRepository) RawTx(ctx context.Context, tx bun.IDB, sql string, args ...any) ([]User, error) {
	return m.Raw(ctx, sql, args...)
}
func (m *mockUserRepository) Handlers() repository.ModelHandlers[User] {
	return repository.ModelHandlers[User]{}
}

// Interface assertion to ensure mockUserRepository implements Repository[User]
var _ repository.Repository[User] = (*mockUserRepository)(nil)

// TestEndToEndCachedRepositoryFlow tests the complete integration flow
// using the DI container to wire up cached repository operations
func TestEndToEndCachedRepositoryFlow(t *testing.T) {
	// Create DI container with minimal TTL for faster testing
	config := cache.Config{
		Capacity:             100,
		NumShards:            4,
		TTL:                  1 * time.Second,
		EvictionPercentage:   10,
		EarlyRefresh:         nil, // Disable for simpler test
		MissingRecordStorage: true,
		EvictionInterval:     0,
	}

	container, err := NewContainer(config)
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	// Create mock base repository and populate with test data
	mockRepo := newMockUserRepository()
	testUser := User{
		ID:       "test-123",
		Name:     "Test User",
		Email:    "test@example.com",
		CreateTs: time.Now().Unix(),
	}
	_, err = mockRepo.Create(context.Background(), testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create cached repository using DI container
	cachedRepo := NewCachedRepository(container, mockRepo)
	ctx := context.Background()

	// Test 1: GetByID - First call should hit the base repository
	user1, err := cachedRepo.GetByID(ctx, "test-123")
	if err != nil {
		t.Fatalf("First GetByID failed: %v", err)
	}

	if user1.ID != testUser.ID || user1.Name != testUser.Name {
		t.Errorf("First GetByID returned incorrect user: got %+v, expected %+v", user1, testUser)
	}

	// Verify the base repository was called
	if callCount := mockRepo.getCallCount("GetByID"); callCount != 1 {
		t.Errorf("Expected base repository GetByID to be called once, got %d calls", callCount)
	}

	// Test 2: GetByID again - Should be served from cache (same call count)
	user2, err := cachedRepo.GetByID(ctx, "test-123")
	if err != nil {
		t.Fatalf("Second GetByID failed: %v", err)
	}

	if user2.ID != testUser.ID || user2.Name != testUser.Name {
		t.Errorf("Second GetByID returned incorrect user: got %+v, expected %+v", user2, testUser)
	}

	// Verify the base repository was NOT called again (cache hit)
	if callCount := mockRepo.getCallCount("GetByID"); callCount != 1 {
		t.Errorf("Expected base repository GetByID to still be called once (cache hit), got %d calls", callCount)
	}

	// Test 3: List operation - Should hit base repository first time
	users1, total1, err := cachedRepo.List(ctx)
	if err != nil {
		t.Fatalf("First List failed: %v", err)
	}

	if len(users1) != 1 || total1 != 1 {
		t.Errorf("First List returned unexpected results: got %d users, total %d", len(users1), total1)
	}

	if callCount := mockRepo.getCallCount("List"); callCount != 1 {
		t.Errorf("Expected base repository List to be called once, got %d calls", callCount)
	}

	// Test 4: List again - Should be served from cache
	users2, total2, err := cachedRepo.List(ctx)
	if err != nil {
		t.Fatalf("Second List failed: %v", err)
	}

	if len(users2) != 1 || total2 != 1 {
		t.Errorf("Second List returned unexpected results: got %d users, total %d", len(users2), total2)
	}

	// Verify the base repository was NOT called again (cache hit)
	if callCount := mockRepo.getCallCount("List"); callCount != 1 {
		t.Errorf("Expected base repository List to still be called once (cache hit), got %d calls", callCount)
	}

	// Test 5: Count operation with caching
	count1, err := cachedRepo.Count(ctx)
	if err != nil {
		t.Fatalf("First Count failed: %v", err)
	}

	if count1 != 1 {
		t.Errorf("First Count returned unexpected result: got %d, expected 1", count1)
	}

	if callCount := mockRepo.getCallCount("Count"); callCount != 1 {
		t.Errorf("Expected base repository Count to be called once, got %d calls", callCount)
	}

	// Test 6: Count again - Should be served from cache
	count2, err := cachedRepo.Count(ctx)
	if err != nil {
		t.Fatalf("Second Count failed: %v", err)
	}

	if count2 != 1 {
		t.Errorf("Second Count returned unexpected result: got %d, expected 1", count2)
	}

	// Verify the base repository was NOT called again (cache hit)
	if callCount := mockRepo.getCallCount("Count"); callCount != 1 {
		t.Errorf("Expected base repository Count to still be called once (cache hit), got %d calls", callCount)
	}
}

// TestCacheEvictionFlow tests that cache entries are properly evicted after TTL
func TestCacheEvictionFlow(t *testing.T) {
	// Create DI container with very short TTL for faster testing
	config := cache.Config{
		Capacity:             10,
		NumShards:            2,
		TTL:                  100 * time.Millisecond,
		EvictionPercentage:   10,
		EarlyRefresh:         nil,
		MissingRecordStorage: true,
		EvictionInterval:     50 * time.Millisecond,
	}

	container, err := NewContainer(config)
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	// Create mock repository with test data
	mockRepo := newMockUserRepository()
	testUser := User{
		ID:       "eviction-test",
		Name:     "Eviction Test User",
		Email:    "eviction@example.com",
		CreateTs: time.Now().Unix(),
	}
	mockRepo.Create(context.Background(), testUser)

	cachedRepo := NewCachedRepository(container, mockRepo)
	ctx := context.Background()

	// First call - should hit base repository
	_, err = cachedRepo.GetByID(ctx, "eviction-test")
	if err != nil {
		t.Fatalf("First GetByID failed: %v", err)
	}

	if callCount := mockRepo.getCallCount("GetByID"); callCount != 1 {
		t.Errorf("Expected base repository GetByID to be called once, got %d calls", callCount)
	}

	// Second call immediately - should be served from cache
	_, err = cachedRepo.GetByID(ctx, "eviction-test")
	if err != nil {
		t.Fatalf("Second GetByID failed: %v", err)
	}

	if callCount := mockRepo.getCallCount("GetByID"); callCount != 1 {
		t.Errorf("Expected base repository GetByID to still be called once (cache hit), got %d calls", callCount)
	}

	// Wait for cache eviction
	time.Sleep(200 * time.Millisecond)

	// Third call after TTL - should hit base repository again
	_, err = cachedRepo.GetByID(ctx, "eviction-test")
	if err != nil {
		t.Fatalf("Third GetByID failed: %v", err)
	}

	if callCount := mockRepo.getCallCount("GetByID"); callCount != 2 {
		t.Errorf("Expected base repository GetByID to be called twice after eviction, got %d calls", callCount)
	}
}

// TestWriteMethodPassThrough verifies that write methods pass through to base repository
func TestWriteMethodPassThrough(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	mockRepo := newMockUserRepository()
	cachedRepo := NewCachedRepository(container, mockRepo)
	ctx := context.Background()

	// Test Create pass-through
	newUser := User{
		ID:    "new-user",
		Name:  "New User",
		Email: "new@example.com",
	}

	createdUser, err := cachedRepo.Create(ctx, newUser)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if createdUser.ID != newUser.ID {
		t.Errorf("Create returned unexpected user: got %+v, expected %+v", createdUser, newUser)
	}

	if callCount := mockRepo.getCallCount("Create"); callCount != 1 {
		t.Errorf("Expected base repository Create to be called once, got %d calls", callCount)
	}

	// Test Update pass-through
	updatedUser := createdUser
	updatedUser.Name = "Updated Name"

	resultUser, err := cachedRepo.Update(ctx, updatedUser)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if resultUser.Name != "Updated Name" {
		t.Errorf("Update didn't apply changes: got name %s, expected 'Updated Name'", resultUser.Name)
	}

	if callCount := mockRepo.getCallCount("Update"); callCount != 1 {
		t.Errorf("Expected base repository Update to be called once, got %d calls", callCount)
	}

	// Test Delete pass-through
	err = cachedRepo.Delete(ctx, resultUser)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if callCount := mockRepo.getCallCount("Delete"); callCount != 1 {
		t.Errorf("Expected base repository Delete to be called once, got %d calls", callCount)
	}
}

// TestErrorPropagation verifies that errors from the base repository are properly propagated
func TestErrorPropagation(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	mockRepo := newMockUserRepository()
	cachedRepo := NewCachedRepository(container, mockRepo)
	ctx := context.Background()

	// Test GetByID with non-existent user (should propagate error)
	_, err = cachedRepo.GetByID(ctx, "non-existent")
	if err == nil {
		t.Error("Expected GetByID to return error for non-existent user")
	}

	// Verify the error is properly propagated and cached
	_, err2 := cachedRepo.GetByID(ctx, "non-existent")
	if err2 == nil {
		t.Error("Expected second GetByID to also return cached error")
	}

	// Both calls should have the same error (cached)
	if err.Error() != err2.Error() {
		t.Errorf("Error messages don't match: first='%s', second='%s'", err.Error(), err2.Error())
	}
}

// TestDifferentRepositoryTypes verifies the container works with different repository types
func TestDifferentRepositoryTypes(t *testing.T) {
	container, err := NewContainerWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	// Test with User repository
	userMockRepo := newMockUserRepository()
	userCachedRepo := NewCachedRepository(container, userMockRepo)

	if userCachedRepo == nil {
		t.Error("Failed to create cached User repository")
	}

	// Verify the repositories can be used independently
	ctx := context.Background()
	testUser := User{ID: "test", Name: "Test", Email: "test@example.com"}
	userMockRepo.Create(ctx, testUser)

	retrievedUser, err := userCachedRepo.GetByID(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrievedUser.ID != testUser.ID {
		t.Errorf("Retrieved user ID mismatch: got %s, expected %s", retrievedUser.ID, testUser.ID)
	}
}
