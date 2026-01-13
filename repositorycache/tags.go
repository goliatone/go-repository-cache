package repositorycache

import (
	"context"
)

type cacheTagsContextKey struct{}

// WithCacheTags attaches additional cache tags to the context for read registration.
func WithCacheTags(ctx context.Context, tags ...string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(tags) == 0 {
		return ctx
	}

	existing := cacheTagsFromContext(ctx)
	combined := append(existing, tags...)
	combined = dedupeStrings(combined)
	if len(combined) == 0 {
		return ctx
	}

	return context.WithValue(ctx, cacheTagsContextKey{}, combined)
}

func cacheTagsFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	if tags, ok := ctx.Value(cacheTagsContextKey{}).([]string); ok {
		return append([]string(nil), tags...)
	}
	return nil
}
