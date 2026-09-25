package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type validateRemoteFilePath struct{ bundle.RO }

func (v *validateRemoteFilePath) Name() string {
	return "validate:remote_file_path"
}

func (v *validateRemoteFilePath) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Under immutable_folder, file_path is not a sync destination: it is replaced
	// with the content-addressed snapshot path at deploy time.
	if b.IsImmutableFolder() {
		return nil
	}

	// Nothing to check if the user opted out of syncing. Mirrors FilesToSync.
	if len(b.Config.Sync.Paths) == 0 || b.Config.Workspace.FilePath == "" {
		return nil
	}

	var me *iam.User
	if b.Config.Workspace.CurrentUser != nil {
		me = b.Config.Workspace.CurrentUser.User
	}

	// dryRun=true: assert the destination is a usable directory/repo without
	// creating it. Deploy still creates it via sync.New.
	err := sync.EnsureRemotePathIsUsable(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.FilePath, me, true)
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   err.Error(),
				Locations: b.Config.GetLocations("workspace.file_path"),
				Paths:     []dyn.Path{dyn.MustPathFromString("workspace.file_path")},
			},
		}
	}

	return nil
}

func ValidateRemoteFilePath() bundle.ReadOnlyMutator {
	return &validateRemoteFilePath{}
}
