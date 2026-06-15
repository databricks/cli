package statemgmt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/direct/dstate"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// LoadStateFromDMS populates the in-memory DeploymentState DB from the
// deployment metadata service for the deployment identified by
// b.DeploymentID. The state is opened in read mode via OpenWithData; the
// historical local resources.json file is never touched on the DMS path.
//
// The path passed to OpenWithData is the local resources.json path: it is
// only used for diagnostics and as the eventual write target if the deploy
// path later upgrades the state to write mode (step 5 territory; today no
// callers do that under DMS).
//
// When b.DeploymentID is empty the function is a no-op: this is the
// "DMS enabled but no prior deployment" case, where the state is genuinely
// empty until the lock package creates the deployment.
func LoadStateFromDMS(ctx context.Context, b *bundle.Bundle) error {
	if b.DeploymentID == "" {
		// Initialize an empty state so subsequent reads (e.g. ExportState)
		// don't panic on an unopened DB.
		_, localPath := b.StateFilenameDirect(ctx)
		b.DeploymentBundle.StateDB.OpenWithData(localPath, dstate.NewDatabase("", 0))
		return nil
	}

	w := b.WorkspaceClient(ctx)
	resources, err := w.Bundle.ListResourcesAll(ctx, sdkbundle.ListResourcesRequest{
		Parent: "deployments/" + b.DeploymentID,
	})
	if err != nil {
		return fmt.Errorf("failed to list resources from deployment metadata service: %w", err)
	}

	data := dstate.NewDatabase("", 0)
	for _, r := range resources {
		// DMS reports resource keys without the "resources." prefix (e.g.
		// "jobs.foo"); the local state DB uses the fully-qualified form
		// ("resources.jobs.foo") as its map key, so prepend it here.
		stateKey := "resources." + r.ResourceKey

		var stateBytes json.RawMessage
		if r.State != nil {
			stateBytes = *r.State
		}

		data.State[stateKey] = dstate.ResourceEntry{
			ID:    r.ResourceId,
			State: stateBytes,
		}
	}

	_, localPath := b.StateFilenameDirect(ctx)
	b.DeploymentBundle.StateDB.OpenWithData(localPath, data)
	return nil
}
