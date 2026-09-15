package dresources

import "context"

type resourceKeyType struct{}

// WithResourceKey attaches the bundle resource key used in cmdio progress lines.
// key is the plan form without the "resources." prefix (e.g. "job_runs.foo").
func WithResourceKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, resourceKeyType{}, key)
}

// ResourceKey returns the key attached by [WithResourceKey], or "" if none.
func ResourceKey(ctx context.Context) string {
	key, _ := ctx.Value(resourceKeyType{}).(string)
	return key
}
