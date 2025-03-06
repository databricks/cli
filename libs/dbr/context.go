package dbr

import "context"

// key is a package-local type to use for context keys.
//
// Using an unexported type for context keys prevents key collisions across
// packages since external packages cannot create values of this type.
type key int

const (
	// dbrKey is the context key for the detection result.
	// The value of 1 is arbitrary and can be any number.
	// Other keys in the same package must have different values.
	dbrKey = key(1)
)

type Environment struct {
	IsDbr   bool
	Version string
}

// DetectRuntime detects whether or not the current
// process is running inside a Databricks Runtime environment.
// It return a new context with the detection result set.
func DetectRuntime(ctx context.Context) context.Context {
	if v := ctx.Value(dbrKey); v != nil {
		panic("dbr.DetectRuntime called twice on the same context")
	}
	return context.WithValue(ctx, dbrKey, detect(ctx))
}

// MockRuntime is a helper function to mock the detection result.
// It returns a new context with the detection result set.
func MockRuntime(ctx context.Context, runtime Environment) context.Context {
	if v := ctx.Value(dbrKey); v != nil {
		panic("dbr.MockRuntime called twice on the same context")
	}
	return context.WithValue(ctx, dbrKey, runtime)
}

// RunsOnRuntime returns the detection result from the context.
// It expects a context returned by [DetectRuntime] or [MockRuntime].
//
// We store this value in a context to avoid having to use either
// a global variable, passing a boolean around everywhere, or
// performing the same detection multiple times.
func RunsOnRuntime(ctx context.Context) bool {
	v := ctx.Value(dbrKey)
	if v == nil {
		panic("dbr.RunsOnRuntime called without calling dbr.DetectRuntime first")
	}
	return v.(Environment).IsDbr
}

func RuntimeVersion(ctx context.Context) string {
	v := ctx.Value(dbrKey)
	if v == nil {
		panic("dbr.RuntimeVersion called without calling dbr.DetectRuntime first")
	}

	return v.(Environment).Version
}
