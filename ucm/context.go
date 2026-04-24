package ucm

import (
	"context"
)

// Placeholder to use as unique key in context.Context.
var ucmKey int

// Context stores the specified ucm on a new context.
// The ucm is available through the `Get()` function.
func Context(ctx context.Context, u *Ucm) context.Context {
	return context.WithValue(ctx, &ucmKey, u)
}

// GetOrNil returns the ucm as configured on the context.
// It returns nil if it isn't configured.
func GetOrNil(ctx context.Context) *Ucm {
	ucm, ok := ctx.Value(&ucmKey).(*Ucm)
	if !ok {
		return nil
	}
	return ucm
}

// Get returns the ucm as configured on the context.
// It panics if it isn't configured.
func Get(ctx context.Context) *Ucm {
	u := GetOrNil(ctx)
	if u == nil {
		panic("context not configured with ucm")
	}
	return u
}
