package dresources

import "context"

type lifecycleStartedKeyType struct{}

// WithLifecycleStarted returns a context with lifecycle.started set to true.
func WithLifecycleStarted(ctx context.Context) context.Context {
	return context.WithValue(ctx, lifecycleStartedKeyType{}, true)
}

// lifecycleStartedFromContext returns true if lifecycle.started is set in the context.
func lifecycleStartedFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(lifecycleStartedKeyType{}).(bool)
	return v
}
