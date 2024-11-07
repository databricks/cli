package dbr

import "context"

// key is a private type to prevent collisions with other packages.
type key int

const (
	// dbrKey is the context key for the detection result.
	// The value of 1 is arbitrary and can be any number.
	// Other keys in the same package must have different values.
	dbrKey = key(1)
)

// DetectRuntime detects whether or not the current
// process is running inside a Databricks Runtime environment.
// It return a new context with the detection result cached.
func DetectRuntime(ctx context.Context) context.Context {
	if v := ctx.Value(dbrKey); v != nil {
		panic("dbr.DetectRuntime called twice on the same context")
	}
	return context.WithValue(ctx, dbrKey, detect(ctx))
}

// MockRuntime is a helper function to mock the detection result.
// It returns a new context with the detection result cached.
func MockRuntime(ctx context.Context, b bool) context.Context {
	if v := ctx.Value(dbrKey); v != nil {
		panic("dbr.MockRuntime called twice on the same context")
	}
	return context.WithValue(ctx, dbrKey, b)
}

// RunsOnRuntime returns the cached detection result from the context.
// It expects a context returned by [DetectRuntime] or [MockRuntime].
func RunsOnRuntime(ctx context.Context) bool {
	v := ctx.Value(dbrKey)
	if v == nil {
		panic("dbr.RunsOnRuntime called without calling dbr.DetectRuntime first")
	}
	return v.(bool)
}
