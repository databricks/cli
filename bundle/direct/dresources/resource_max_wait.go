package dresources

import (
	"context"
	"time"
)

type resourceMaxWaitType struct{}

// WithResourceMaxWait attaches the engine's already-resolved
// DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT cap so a resource wait can bound itself by it
// without re-reading and re-parsing the environment on every call. A non-positive
// duration means "no cap".
func WithResourceMaxWait(ctx context.Context, maxWait time.Duration) context.Context {
	return context.WithValue(ctx, resourceMaxWaitType{}, maxWait)
}

// resourceMaxWait returns the cap attached by [WithResourceMaxWait]; ok is false when
// none was attached.
func resourceMaxWait(ctx context.Context) (maxWait time.Duration, ok bool) {
	maxWait, ok = ctx.Value(resourceMaxWaitType{}).(time.Duration)
	return maxWait, ok
}
