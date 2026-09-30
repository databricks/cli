package bundle

import (
	"context"
)

// Placeholder to use as unique key in context.Context.
var bundleKey int

// GetOrNil returns the bundle as configured on the context.
// It returns nil if it isn't configured.
func GetOrNil(ctx context.Context) *Bundle {
	bundle, ok := ctx.Value(&bundleKey).(*Bundle)
	if !ok {
		return nil
	}
	return bundle
}
