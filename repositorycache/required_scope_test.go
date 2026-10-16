package repositorycache

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	repository "github.com/goliatone/go-repository-bun"
	"github.com/goliatone/go-repository-cache/cache"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

type cacheScopeResource struct {
	bun.BaseModel `bun:"table:cache_scope_resources,alias:r"`
	ID            uuid.UUID `bun:"id,pk"`
	GroupID       string    `bun:"group_id,notnull"`
	Name          string    `bun:"name,unique,notnull"`
	Value         string    `bun:"value,notnull"`
}

type cacheScopeQueryCounter struct{ calls atomic.Int64 }

func (c *cacheScopeQueryCounter) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	c.calls.Add(1)
	return ctx
}

func (*cacheScopeQueryCounter) AfterQuery(context.Context, *bun.QueryEvent) {}

func newCacheScopeFixture(t *testing.T, defaultsBeforeWrapping bool) (*CachedRepository[*cacheScopeResource], *cacheScopeQueryCounter, []*cacheScopeResource) {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "scope.db"))+"?_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*cacheScopeResource)(nil)).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	resources := []*cacheScopeResource{
		{ID: uuid.New(), GroupID: "Group A", Name: "resource-a1", Value: "opaque-a1"},
		{ID: uuid.New(), GroupID: "Group A", Name: "resource-a2", Value: "opaque-a2"},
		{ID: uuid.New(), GroupID: "Group B", Name: "resource-b", Value: "opaque-b"},
	}
	if _, err := db.NewInsert().Model(&resources).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	base := repository.MustNewRepository(db, repository.ModelHandlers[*cacheScopeResource]{
		NewRecord:          func() *cacheScopeResource { return &cacheScopeResource{} },
		GetID:              func(r *cacheScopeResource) uuid.UUID { return r.ID },
		SetID:              func(r *cacheScopeResource, id uuid.UUID) { r.ID = id },
		GetIdentifier:      func() string { return "name" },
		GetIdentifierValue: func(r *cacheScopeResource) string { return r.Name },
	})
	defaults := repository.ScopeDefaults{Select: []string{"group"}}
	if defaultsBeforeWrapping {
		base.RegisterScope("group", repository.ScopeByFieldRequired("group", "group_id"))
		if err := base.SetScopeDefaults(defaults); err != nil {
			t.Fatal(err)
		}
	}
	service, err := cache.NewCacheService(cache.Config{
		Capacity: 256, NumShards: 1, TTL: time.Minute, EvictionPercentage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	cached := New(base, service, cache.NewDefaultKeySerializer())
	if !defaultsBeforeWrapping {
		cached.RegisterScope("group", repository.ScopeByFieldRequired("group", "group_id"))
		if err := cached.SetScopeDefaults(defaults); err != nil {
			t.Fatal(err)
		}
	}
	queries := &cacheScopeQueryCounter{}
	db.AddQueryHook(queries)
	return cached, queries, resources
}

// Check the public results and observe SQL calls, rather than asserting a
// particular cache key representation. Distinct group sizes expose Count leaks.
func cacheScopeRead(t *testing.T, cached *CachedRepository[*cacheScopeResource], operation string, ctx context.Context, target *cacheScopeResource, wantGroup string, wantTotal int, criteria ...repository.SelectCriteria) {
	t.Helper()
	var rows []*cacheScopeResource
	var total int
	var err error
	switch operation {
	case "Get", "GetByID", "GetByIdentifier":
		var row *cacheScopeResource
		switch operation {
		case "Get":
			row, err = cached.Get(ctx, criteria...)
		case "GetByID":
			row, err = cached.GetByID(ctx, target.ID.String(), criteria...)
		case "GetByIdentifier":
			row, err = cached.GetByIdentifier(ctx, target.Name, criteria...)
		}
		if wantTotal == 0 {
			if !repository.IsRecordNotFound(err) || row != nil {
				t.Fatalf("%s: want not-found, got row=%+v err=%v", operation, row, err)
			}
			return
		}
		if row != nil {
			if operation != "Get" && row.ID != target.ID {
				t.Fatalf("%s returned ID=%s want=%s", operation, row.ID, target.ID)
			}
			rows = []*cacheScopeResource{row}
			total = 1
		}
		wantTotal = 1
	case "List":
		rows, total, err = cached.List(ctx, criteria...)
	case "Count":
		total, err = cached.Count(ctx, criteria...)
	default:
		t.Fatalf("unknown operation %q", operation)
	}
	if err != nil {
		t.Fatalf("%s: %v", operation, err)
	}
	if total != wantTotal {
		t.Fatalf("%s: total=%d want=%d", operation, total, wantTotal)
	}
	if operation != "Count" && len(rows) != wantTotal {
		t.Fatalf("%s: rows=%d want=%d", operation, len(rows), wantTotal)
	}
	for _, row := range rows {
		if row.GroupID != wantGroup {
			t.Fatalf("%s leaked resource: %+v; want %s", operation, row, wantGroup)
		}
	}
}

func TestCachedRepository_RequiredScopeWarmCache(t *testing.T) {
	for _, config := range []struct {
		name   string
		before bool
	}{{"defaults_before_wrapping", true}, {"defaults_through_decorator", false}} {
		t.Run(config.name, func(t *testing.T) {
			for _, operation := range []string{"Get", "GetByID", "GetByIdentifier", "List", "Count"} {
				t.Run(operation, func(t *testing.T) {
					cached, queries, resources := newCacheScopeFixture(t, config.before)
					ctxA := repository.WithScopeData(context.Background(), "group", "Group A")
					readA := func() { cacheScopeRead(t, cached, operation, ctxA, resources[0], "Group A", 2) }
					readA()
					warmQueries := queries.calls.Load()
					if warmQueries == 0 {
						t.Fatal("initial read did not reach SQLite")
					}
					readA()
					if queries.calls.Load() != warmQueries {
						t.Fatal("repeated Group A read did not hit the cache")
					}
					for _, denied := range []struct {
						name string
						ctx  context.Context
					}{
						{"missing", context.Background()},
						{"wrong_group", repository.WithScopeData(context.Background(), "group", "Group C")},
						{"unregistered", repository.WithSelectScopes(ctxA, "unregistered-group")},
					} {
						t.Run(denied.name, func(t *testing.T) {
							before := queries.calls.Load()
							cacheScopeRead(t, cached, operation, denied.ctx, resources[0], "", 0)
							if queries.calls.Load() <= before {
								t.Fatal("distinct denied context reused a warmed result instead of reaching SQLite")
							}
							cacheScopeRead(t, cached, operation, denied.ctx, resources[0], "", 0)
						})
					}
					ctxB := repository.WithScopeData(context.Background(), "group", "Group B")
					// The same Group A ID/identifier cannot be read in Group B.
					wantB := 1
					if operation == "GetByID" || operation == "GetByIdentifier" {
						wantB = 0
					}
					cacheScopeRead(t, cached, operation, ctxB, resources[0], "Group B", wantB)
					cacheScopeRead(t, cached, operation, ctxB, resources[2], "Group B", 1)
					before := queries.calls.Load()
					readA()
					if queries.calls.Load() != before {
						t.Fatal("other contexts displaced the valid Group A cache entry")
					}
				})
			}
		})
	}
}

func TestCachedRepository_RequiredScopeCriteriaBypassWarmCache(t *testing.T) {
	for _, operation := range []string{"Get", "GetByID", "GetByIdentifier", "List", "Count"} {
		t.Run(operation, func(t *testing.T) {
			cached, queries, resources := newCacheScopeFixture(t, true)
			ctx := repository.WithScopeData(context.Background(), "group", "Group A")
			cacheScopeRead(t, cached, operation, ctx, resources[0], "Group A", 2)
			warmQueries := queries.calls.Load()
			cacheScopeRead(t, cached, operation, ctx, resources[0], "Group A", 2)
			if queries.calls.Load() != warmQueries {
				t.Fatal("expected a warm cache before criteria reads")
			}
			for attempt := 0; attempt < 2; attempt++ {
				before := queries.calls.Load()
				cacheScopeRead(t, cached, operation, ctx, resources[0], "", 0, repository.SelectBy("value", "=", "absent-payload"))
				if queries.calls.Load() <= before {
					t.Fatal("criteria read did not reach SQLite")
				}
				before = queries.calls.Load()
				// An ordinary caller OR must not bypass the required Group scope.
				wantTotal := 1
				if operation == "GetByID" || operation == "GetByIdentifier" {
					// A direct lookup of B remains not-found even when caller OR
					// criteria mention both A and B.
					wantTotal = 0
				}
				cacheScopeRead(t, cached, operation, ctx, resources[2], "Group A", wantTotal,
					repository.SelectByID(resources[0].ID.String()),
					repository.SelectOrBy("id", "=", resources[2].ID.String()))
				if queries.calls.Load() <= before {
					t.Fatal("OR criteria read did not reach SQLite")
				}
			}
			before := queries.calls.Load()
			cacheScopeRead(t, cached, operation, ctx, resources[0], "Group A", 2)
			if queries.calls.Load() != before {
				t.Fatal("criteria reads changed the unfiltered cached result")
			}
		})
	}
}
