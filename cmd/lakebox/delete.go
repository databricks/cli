package lakebox

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <lakebox-id>",
		Short: "Delete a Lakebox environment",
		Long: `Delete a Lakebox environment.

Permanently terminates and removes the specified lakebox.

Example:
  databricks lakebox delete happy-panda-1234`,
		Args:              cobra.ExactArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSandboxIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			lakeboxID := args[0]
			s := spin(ctx, "Removing "+lakeboxID+"…")
			defer s.Close()

			if err := api.delete(ctx, lakeboxID); err != nil {
				s.fail("Failed to delete " + lakeboxID)
				return fmt.Errorf("failed to delete lakebox %s: %w", lakeboxID, err)
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			if getDefault(ctx, profile) == lakeboxID {
				_ = clearDefault(ctx, profile)
				s.ok("Removed " + cmdio.Bold(ctx, lakeboxID) + " " + cmdio.Dim(ctx, "(default cleared)"))
			} else {
				s.ok("Removed " + cmdio.Bold(ctx, lakeboxID))
			}
			return nil
		},
	}

	return cmd
}
