package dresources

import "context"

// CreateIdentity identifies the resource being created, for resources that
// derive a stable create-time key from it (e.g. a run-now idempotency token).
//
// Every part is reconstructible from config and state, so a deploy that crashes
// mid-create derives the same identity on the next attempt.
type CreateIdentity struct {
	// Deployment is the bundle's workspace root path. It keeps deployments that
	// share a workspace (different targets, different users) apart.
	Deployment string

	// ResourceKey is the deployment-local key, e.g. "resources.job_runs.nightly".
	ResourceKey string

	// PriorID is the id of a resource whose remote copy has vanished, set only
	// when re-creating it. Folding it in keeps the new key clear of the gone one,
	// which the backend may still hold (e.g. a tombstoned run token).
	PriorID string
}

type createIdentityKey struct{}

// WithCreateIdentity records the identity of the resource being created. Set by
// the framework in DeploymentUnit.Deploy; see CreateIdentity.
func WithCreateIdentity(ctx context.Context, id CreateIdentity) context.Context {
	return context.WithValue(ctx, createIdentityKey{}, id)
}

// GetCreateIdentity returns the identity set by WithCreateIdentity. ok is false
// outside a deploy, e.g. in tests that drive a resource directly.
func GetCreateIdentity(ctx context.Context) (CreateIdentity, bool) {
	id, ok := ctx.Value(createIdentityKey{}).(CreateIdentity)
	return id, ok
}
