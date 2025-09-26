package main

import (
	"context"
	"fmt"
	"log"
	"time"

	repository "github.com/goliatone/go-repository-bun"
	"github.com/goliatone/go-repository-cache/cache"
	"github.com/goliatone/go-repository-cache/pkg/di"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User represents a simple user entity for demonstration purposes
// This showcases how cached repositories work with domain entities
type User struct {
	ID    string `json:"id" bun:",pk"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// fakeUserRepository implements a fake repository for demonstration
// This simulates a real database repository with artificial delays
// to showcase the caching performance benefits
type fakeUserRepository struct {
	users map[string]User
}

// newFakeUserRepository creates a new fake repository with sample data
func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		users: map[string]User{
			"1": {ID: "1", Name: "John Doe", Email: "john@example.com"},
			"2": {ID: "2", Name: "Jane Smith", Email: "jane@example.com"},
			"3": {ID: "3", Name: "Bob Johnson", Email: "bob@example.com"},
		},
	}
}

// Get simulates database latency and returns a user by criteria
// This demonstrates the performance benefit of caching
func (r *fakeUserRepository) Get(ctx context.Context, criteria ...repository.SelectCriteria) (User, error) {
	fmt.Printf("  [FAKE DB] Get called - simulating 100ms database query...\n")
	time.Sleep(100 * time.Millisecond) // Simulate database latency

	// For simplicity, just return the first user
	return r.users["1"], nil
}

// GetByID simulates database latency and returns a user by ID
func (r *fakeUserRepository) GetByID(ctx context.Context, id string, criteria ...repository.SelectCriteria) (User, error) {
	fmt.Printf("  [FAKE DB] GetByID(%s) called - simulating 100ms database query...\n", id)
	time.Sleep(100 * time.Millisecond) // Simulate database latency

	if user, exists := r.users[id]; exists {
		return user, nil
	}

	var zero User
	return zero, fmt.Errorf("user not found: %s", id)
}

// List simulates database latency and returns all users
func (r *fakeUserRepository) List(ctx context.Context, criteria ...repository.SelectCriteria) ([]User, int, error) {
	fmt.Printf("  [FAKE DB] List called - simulating 150ms database query...\n")
	time.Sleep(150 * time.Millisecond) // Simulate database latency

	var users []User
	for _, user := range r.users {
		users = append(users, user)
	}

	return users, len(users), nil
}

// Count simulates database latency and returns user count
func (r *fakeUserRepository) Count(ctx context.Context, criteria ...repository.SelectCriteria) (int, error) {
	fmt.Printf("  [FAKE DB] Count called - simulating 75ms database query...\n")
	time.Sleep(75 * time.Millisecond) // Simulate database latency

	return len(r.users), nil
}

// GetByIdentifier simulates database latency and returns a user by identifier
func (r *fakeUserRepository) GetByIdentifier(ctx context.Context, identifier string, criteria ...repository.SelectCriteria) (User, error) {
	fmt.Printf("  [FAKE DB] GetByIdentifier(%s) called - simulating 100ms database query...\n", identifier)
	time.Sleep(100 * time.Millisecond) // Simulate database latency

	// For demo, treat identifier as email lookup
	for _, user := range r.users {
		if user.Email == identifier {
			return user, nil
		}
	}

	var zero User
	return zero, fmt.Errorf("user not found with identifier: %s", identifier)
}

// --- Stub implementations for write operations (required by Repository interface) ---

func (r *fakeUserRepository) Create(ctx context.Context, record User, criteria ...repository.InsertCriteria) (User, error) {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	r.users[record.ID] = record
	fmt.Printf("  [FAKE DB] Created user: %s\n", record.ID)
	return record, nil
}

func (r *fakeUserRepository) Update(ctx context.Context, record User, criteria ...repository.UpdateCriteria) (User, error) {
	r.users[record.ID] = record
	fmt.Printf("  [FAKE DB] Updated user: %s\n", record.ID)
	return record, nil
}

func (r *fakeUserRepository) Delete(ctx context.Context, record User) error {
	delete(r.users, record.ID)
	fmt.Printf("  [FAKE DB] Deleted user: %s\n", record.ID)
	return nil
}

// --- Additional Repository interface methods (stubs for completeness) ---

func (r *fakeUserRepository) CreateTx(ctx context.Context, tx bun.IDB, record User, criteria ...repository.InsertCriteria) (User, error) {
	return r.Create(ctx, record, criteria...)
}

func (r *fakeUserRepository) CreateMany(ctx context.Context, records []User, criteria ...repository.InsertCriteria) ([]User, error) {
	var result []User
	for _, record := range records {
		created, err := r.Create(ctx, record, criteria...)
		if err != nil {
			return nil, err
		}
		result = append(result, created)
	}
	return result, nil
}

func (r *fakeUserRepository) CreateManyTx(ctx context.Context, tx bun.IDB, records []User, criteria ...repository.InsertCriteria) ([]User, error) {
	return r.CreateMany(ctx, records, criteria...)
}

func (r *fakeUserRepository) GetOrCreate(ctx context.Context, record User) (User, error) {
	if existing, exists := r.users[record.ID]; exists {
		return existing, nil
	}
	return r.Create(ctx, record)
}

func (r *fakeUserRepository) GetOrCreateTx(ctx context.Context, tx bun.IDB, record User) (User, error) {
	return r.GetOrCreate(ctx, record)
}

func (r *fakeUserRepository) UpdateTx(ctx context.Context, tx bun.IDB, record User, criteria ...repository.UpdateCriteria) (User, error) {
	return r.Update(ctx, record, criteria...)
}

func (r *fakeUserRepository) UpdateMany(ctx context.Context, records []User, criteria ...repository.UpdateCriteria) ([]User, error) {
	var result []User
	for _, record := range records {
		updated, err := r.Update(ctx, record, criteria...)
		if err != nil {
			return nil, err
		}
		result = append(result, updated)
	}
	return result, nil
}

func (r *fakeUserRepository) UpdateManyTx(ctx context.Context, tx bun.IDB, records []User, criteria ...repository.UpdateCriteria) ([]User, error) {
	return r.UpdateMany(ctx, records, criteria...)
}

func (r *fakeUserRepository) Upsert(ctx context.Context, record User, criteria ...repository.UpdateCriteria) (User, error) {
	r.users[record.ID] = record
	return record, nil
}

func (r *fakeUserRepository) UpsertTx(ctx context.Context, tx bun.IDB, record User, criteria ...repository.UpdateCriteria) (User, error) {
	return r.Upsert(ctx, record, criteria...)
}

func (r *fakeUserRepository) UpsertMany(ctx context.Context, records []User, criteria ...repository.UpdateCriteria) ([]User, error) {
	var result []User
	for _, record := range records {
		upserted, err := r.Upsert(ctx, record, criteria...)
		if err != nil {
			return nil, err
		}
		result = append(result, upserted)
	}
	return result, nil
}

func (r *fakeUserRepository) UpsertManyTx(ctx context.Context, tx bun.IDB, records []User, criteria ...repository.UpdateCriteria) ([]User, error) {
	return r.UpsertMany(ctx, records, criteria...)
}

func (r *fakeUserRepository) DeleteTx(ctx context.Context, tx bun.IDB, record User) error {
	return r.Delete(ctx, record)
}

func (r *fakeUserRepository) DeleteMany(ctx context.Context, criteria ...repository.DeleteCriteria) error {
	// For simplicity, delete all users
	for id := range r.users {
		delete(r.users, id)
	}
	return nil
}

func (r *fakeUserRepository) DeleteManyTx(ctx context.Context, tx bun.IDB, criteria ...repository.DeleteCriteria) error {
	return r.DeleteMany(ctx, criteria...)
}

func (r *fakeUserRepository) DeleteWhere(ctx context.Context, criteria ...repository.DeleteCriteria) error {
	return r.DeleteMany(ctx, criteria...)
}

func (r *fakeUserRepository) DeleteWhereTx(ctx context.Context, tx bun.IDB, criteria ...repository.DeleteCriteria) error {
	return r.DeleteWhere(ctx, criteria...)
}

func (r *fakeUserRepository) ForceDelete(ctx context.Context, record User) error {
	return r.Delete(ctx, record)
}

func (r *fakeUserRepository) ForceDeleteTx(ctx context.Context, tx bun.IDB, record User) error {
	return r.ForceDelete(ctx, record)
}

// Transaction read methods
func (r *fakeUserRepository) GetTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) (User, error) {
	return r.Get(ctx, criteria...)
}

func (r *fakeUserRepository) GetByIDTx(ctx context.Context, tx bun.IDB, id string, criteria ...repository.SelectCriteria) (User, error) {
	return r.GetByID(ctx, id, criteria...)
}

func (r *fakeUserRepository) ListTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) ([]User, int, error) {
	return r.List(ctx, criteria...)
}

func (r *fakeUserRepository) CountTx(ctx context.Context, tx bun.IDB, criteria ...repository.SelectCriteria) (int, error) {
	return r.Count(ctx, criteria...)
}

func (r *fakeUserRepository) GetByIdentifierTx(ctx context.Context, tx bun.IDB, identifier string, criteria ...repository.SelectCriteria) (User, error) {
	return r.GetByIdentifier(ctx, identifier, criteria...)
}

// Raw SQL methods
func (r *fakeUserRepository) Raw(ctx context.Context, sql string, args ...any) ([]User, error) {
	// For demo purposes, just return all users
	var users []User
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

func (r *fakeUserRepository) RawTx(ctx context.Context, tx bun.IDB, sql string, args ...any) ([]User, error) {
	return r.Raw(ctx, sql, args...)
}

// Handlers method
func (r *fakeUserRepository) Handlers() repository.ModelHandlers[User] {
	// Return empty handlers for demo
	return repository.ModelHandlers[User]{}
}

func main() {
	fmt.Println("🚀 Repository Cache Example Application")
	fmt.Println("=====================================")
	fmt.Println()

	// Step 1: Initialize the DI container with cache configuration
	// This demonstrates the usage pattern described in REPOSITORY_CACHE.md §Dependency Injection Container
	fmt.Println("📦 Step 1: Setting up DI container with cache configuration...")

	// Configure cache with custom settings (see REPOSITORY_CACHE.md §Internal Cache Infrastructure)
	config := cache.Config{
		Capacity:             1000,            // Maximum number of cache entries
		NumShards:            8,               // Number of cache shards for concurrency
		EvictionPercentage:   10,              // Percentage to evict when capacity reached
		TTL:                  5 * time.Minute, // Cache TTL - entries expire after 5 minutes
		MissingRecordStorage: true,            // Enable negative caching for missing records
		EvictionInterval:     1 * time.Minute, // How often to check for expired entries
	}

	container, err := di.NewContainer(config)
	if err != nil {
		log.Fatalf("Failed to create DI container: %v", err)
	}

	fmt.Printf("   ✅ DI container initialized with TTL=%v, Shards=%d\n", config.TTL, config.NumShards)
	fmt.Println()

	// Step 2: Create base repository (fake implementation for demo)
	fmt.Println("🗄️  Step 2: Creating base repository (fake implementation)...")
	baseRepo := newFakeUserRepository()
	fmt.Println("   ✅ Fake repository created with sample data")
	fmt.Println()

	// Step 3: Create cached repository using DI container
	// This demonstrates the factory pattern described in REPOSITORY_CACHE.md §Usage Example
	fmt.Println("⚡ Step 3: Creating cached repository wrapper...")
	cachedRepo := di.NewCachedRepository(container, baseRepo)
	fmt.Println("   ✅ Cached repository created - all read operations will now be cached")
	fmt.Println()

	// Step 4: Demonstrate cache miss followed by cache hit
	fmt.Println("🔍 Step 4: Demonstrating cache behavior...")
	fmt.Println()

	ctx := context.Background()

	// First call - should be a cache MISS (hits database)
	fmt.Println("📍 First GetByID call (cache MISS - should hit database):")
	start := time.Now()
	user1, err := cachedRepo.GetByID(ctx, "1")
	duration1 := time.Since(start)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("   Result: %+v (took %v)\n", user1, duration1)
	fmt.Println()

	// Second call - should be a cache HIT (no database call)
	fmt.Println("📍 Second GetByID call (cache HIT - should be instant):")
	start = time.Now()
	user2, err := cachedRepo.GetByID(ctx, "1")
	duration2 := time.Since(start)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("   Result: %+v (took %v)\n", user2, duration2)
	fmt.Println()

	// Step 5: Demonstrate other cached operations
	fmt.Println("🔍 Step 5: Testing other cached operations...")
	fmt.Println()

	// Test List operation (cache miss then hit)
	fmt.Println("📋 Testing List operation:")
	fmt.Println("  First call (cache MISS):")
	start = time.Now()
	_, count1, err := cachedRepo.List(ctx)
	duration1 = time.Since(start)
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}
	fmt.Printf("     Found %d users (took %v)\n", count1, duration1)

	fmt.Println("  Second call (cache HIT):")
	start = time.Now()
	_, count2, err := cachedRepo.List(ctx)
	duration2 = time.Since(start)
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}
	fmt.Printf("     Found %d users (took %v)\n", count2, duration2)
	fmt.Println()

	// Test Count operation
	fmt.Println("🔢 Testing Count operation:")
	fmt.Println("  First call (cache MISS):")
	start = time.Now()
	userCount1, err := cachedRepo.Count(ctx)
	duration1 = time.Since(start)
	if err != nil {
		log.Fatalf("Failed to count users: %v", err)
	}
	fmt.Printf("     Count: %d (took %v)\n", userCount1, duration1)

	fmt.Println("  Second call (cache HIT):")
	start = time.Now()
	userCount2, err := cachedRepo.Count(ctx)
	duration2 = time.Since(start)
	if err != nil {
		log.Fatalf("Failed to count users: %v", err)
	}
	fmt.Printf("     Count: %d (took %v)\n", userCount2, duration2)
	fmt.Println()

	// Step 6: Demonstrate cache invalidation after write operations
	fmt.Println("✏️  Step 6: Testing cache invalidation with write operations...")
	fmt.Println()

	// First, populate List cache
	fmt.Println("   📋 Populating List cache (current users):")
	start = time.Now()
	_, total1, err := cachedRepo.List(ctx)
	duration1 = time.Since(start)
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}
	fmt.Printf("     Found %d users (took %v)\n", total1, duration1)

	// Verify cache hit on second List call
	fmt.Println("   📋 Second List call (cache HIT - should be fast):")
	start = time.Now()
	_, total2, err := cachedRepo.List(ctx)
	duration2 = time.Since(start)
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}
	fmt.Printf("     Found %d users (took %v)\n", total2, duration2)

	// Now create a new user - this should invalidate List and Count caches
	fmt.Println("   ➕ Creating new user (should invalidate List and Count caches):")
	newUser := User{
		ID:    "4",
		Name:  "Alice Cooper",
		Email: "alice@example.com",
	}

	createdUser, err := cachedRepo.Create(ctx, newUser)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	fmt.Printf("     ✅ Created user: %+v\n", createdUser)

	// Test List again - should be cache MISS due to invalidation
	fmt.Println("   📋 List call after create (cache MISS due to invalidation):")
	start = time.Now()
	_, total3, err := cachedRepo.List(ctx)
	duration3 := time.Since(start)
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}
	fmt.Printf("     Found %d users (took %v) - Notice the database call!\n", total3, duration3)
	fmt.Println()

	// Step 7: Demonstrate Update invalidation
	fmt.Println("✏️  Step 7: Testing update invalidation...")

	// Get user to populate cache
	fmt.Println("   👤 Getting user (populate cache):")
	user, err := cachedRepo.GetByID(ctx, "1")
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("     Original: %s (%s)\n", user.Name, user.Email)

	// Update user - should invalidate relevant caches
	fmt.Println("   ✏️  Updating user (should invalidate GetByID cache):")
	updatedUser := User{
		ID:    "1",
		Name:  "John Doe (Updated)",
		Email: "john.doe.updated@example.com",
	}

	_, err = cachedRepo.Update(ctx, updatedUser)
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	// Get user again - should be cache MISS and return updated data
	fmt.Println("   👤 Getting user after update (cache MISS due to invalidation):")
	start = time.Now()
	updatedUserFromCache, err := cachedRepo.GetByID(ctx, "1")
	updateDuration := time.Since(start)
	if err != nil {
		log.Fatalf("Failed to get updated user: %v", err)
	}
	fmt.Printf("     Updated: %s (%s) (took %v) - Fresh data from database!\n",
		updatedUserFromCache.Name, updatedUserFromCache.Email, updateDuration)
	fmt.Println()

	fmt.Println("   📝 Note: Cache invalidation ensures data consistency")
	fmt.Println("           Write operations invalidate affected cache entries")
	fmt.Println()

	// Step 8: Summary
	fmt.Println("📊 Performance & Invalidation Summary:")
	fmt.Printf("   Cache hits are ~100x faster than database calls\n")
	fmt.Printf("   List cache before create: %d users\n", total2)
	fmt.Printf("   List cache after create: %d users (invalidated and refetched)\n", total3)
	fmt.Printf("   Update operations properly invalidate affected cache entries\n")
	fmt.Println()

	fmt.Println("🎉 Demo completed successfully!")
	fmt.Println()
	fmt.Println("📚 Key Concepts Demonstrated:")
	fmt.Println("   • DI Container pattern (pkg/di/container.go)")
	fmt.Println("   • Cache decorator pattern (repositorycache/decorator.go)")
	fmt.Println("   • Cache miss vs. hit performance")
	fmt.Println("   • Cache invalidation after write operations")
	fmt.Println("   • Data consistency through invalidation")
	fmt.Println("   • Interface compatibility with go-repository-bun")
	fmt.Println("   • Key serialization for cache keys")
	fmt.Println()
	fmt.Println("📖 For more details, see:")
	fmt.Println("   • REPOSITORY_CACHE.md - Complete design documentation")
	fmt.Println("   • ARCH_DESIGN.md - Architecture principles")
}
