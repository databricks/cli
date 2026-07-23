package dresources

import "context"

type resourceIdentityKey struct{}

// WithResourceIdentity records the deployment-scoped key of the resource being
// deployed, so resources can derive a stable per-resource seed (e.g. a run-now
// idempotency token). Set once by the framework in DeploymentUnit.Deploy.
func WithResourceIdentity(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, resourceIdentityKey{}, key)
}

// ResourceIdentity returns the key set by WithResourceIdentity, or "" if unset
// (e.g. the unit-test harness that calls resources directly).
func ResourceIdentity(ctx context.Context) string {
	key, _ := ctx.Value(resourceIdentityKey{}).(string)
	return key
}
