package metadata

import (
	"context"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/snapshot"
)

func init() {
	resourceMetadata.Register("internal_immutable_snapshots", computeSnapshots)
}

func computeSnapshots(_ context.Context, b *bundle.Bundle, state resourcestate.ExportedResourcesMap) diag.Diagnostics {
	// For immutable-folder deployments the content lives in a snapshot rather than a regular
	// workspace directory. Record its path so downstream services (e.g. the jobs UI) can locate it.
	// Take the path from the state rather than the config: "bundle deploy --select" can leave the
	// snapshot out, and then the config points at a snapshot that was never uploaded. The
	// snapshot's ID is its relative path.
	if snap, ok := b.Config.Resources.Snapshots[resources.SnapshotResourceKey]; ok {
		if deployed, ok := state[resources.SnapshotKey]; ok {
			b.Metadata.Config.Workspace.SnapshotContentPath = path.Join(snap.RemoteRoot, deployed.ID, snapshot.ContentSubdir)
		}
	}
	return nil
}
