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

type priorResourceIDKey struct{}

// WithPriorResourceID records the id of a resource whose remote copy has
// vanished, letting a re-create derive a fresh identity (e.g. a job run's
// idempotency token) rather than reuse the gone one, which may be tombstoned.
func WithPriorResourceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, priorResourceIDKey{}, id)
}

// PriorResourceID returns the id set by WithPriorResourceID, or "" if unset.
func PriorResourceID(ctx context.Context) string {
	id, _ := ctx.Value(priorResourceIDKey{}).(string)
	return id
}
