package deploy

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/google/uuid"
)

type ensureDeploymentID struct{}

// EnsureDeploymentID loads the local deployment state and ensures
// b.Metrics.DeploymentId is populated before the snapshot upload.
// snapshot.Upload uses it as the bundle_id so the snapshot is keyed to this
// deployment lineage rather than to the bundle name.
// StateUpdate reads the same field back and persists it to disk.
func EnsureDeploymentID() bundle.Mutator {
	return &ensureDeploymentID{}
}

func (*ensureDeploymentID) Name() string { return "deploy:ensure-deployment-id" }

func (*ensureDeploymentID) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Already set (e.g. by a prior call in the same session).
	if b.Metrics.DeploymentId != uuid.Nil {
		return nil
	}

	state, err := load(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	if state.ID == uuid.Nil {
		state.ID = uuid.New()
	}
	b.Metrics.DeploymentId = state.ID
	return nil
}
