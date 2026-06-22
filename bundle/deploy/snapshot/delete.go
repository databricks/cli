package snapshot

import (
	"context"
	"errors"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

type deleteSnapshots struct{}

// DeleteBundleSnapshots removes all snapshots for the current bundle via the
// snapshot API. It does not use workspace.Delete because that requires workspace
// admin rights which non-admin users may not have.
func DeleteBundleSnapshots() bundle.Mutator {
	return &deleteSnapshots{}
}

func (m *deleteSnapshots) Name() string { return "snapshot.DeleteBundleSnapshots" }

func (m *deleteSnapshots) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.SnapshotPath == "" {
		// No snapshot path means no snapshot was ever uploaded (or state was not loaded).
		return nil
	}

	uploader, err := NewSnapshotUploader(b.WorkspaceClient(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	err = uploader.Delete(ctx, b.Config.Bundle.Name)
	if err != nil {
		var apiErr *apierr.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
