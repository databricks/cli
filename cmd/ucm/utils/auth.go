package utils

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

// MustWorkspaceClient is the UCM analogue of root.MustWorkspaceClient. It
// loads ucm.yml, selects the target, resolves a workspace client from
// workspace.host (matching against ~/.databrickscfg) and installs it on the
// command context. Ambiguous matches are resolved interactively, mirroring
// DAB's configureBundle + resolveProfileAmbiguity flow in cmd/root/bundle.go.
func MustWorkspaceClient(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
		cmd.SetContext(ctx)
	}

	u := ProcessUcm(cmd, ProcessOptions{})
	ctx = cmd.Context()
	if u == nil || logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	client, err := u.WorkspaceClientE()
	if err != nil {
		names, isMulti := databrickscfg.AsMultipleProfiles(err)
		if !isMulti {
			logdiag.LogError(ctx, err)
			return root.ErrAlreadyPrinted
		}

		selected, resolveErr := resolveProfileAmbiguity(cmd, u, err, names)
		if resolveErr != nil {
			logdiag.LogError(ctx, resolveErr)
			return root.ErrAlreadyPrinted
		}

		u.Config.Workspace.Profile = selected
		u.ClearWorkspaceClient()
		client, err = u.WorkspaceClientE()
		if err != nil {
			logdiag.LogError(ctx, err)
			return root.ErrAlreadyPrinted
		}
	}

	ctx = cmdctx.SetConfigUsed(ctx, client.Config)
	ctx = cmdctx.SetWorkspaceClient(ctx, client)
	cmd.SetContext(ctx)
	return nil
}

