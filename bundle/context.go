package bundle

import (
	"context"
)

// Placeholder to use as unique key in context.Context.
var bundleKey int

// Context stores the specified bundle on a new context.
// The bundle is available through the `Get()` function.
func Context(ctx context.Context, b *Bundle) context.Context {
	return context.WithValue(ctx, &bundleKey, b)
}

// Get returns the bundle as configured on the context.
// It panics if it isn't configured.
func Get(ctx context.Context) *Bundle {
	bundle, ok := ctx.Value(&bundleKey).(*Bundle)
	if !ok {
		panic("context not configured with bundle")
	}
	return bundle
}
