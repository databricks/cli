package lakebox

import (
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <lakebox-id>",
		Short: "Delete a Lakebox environment",
		Long: `Delete a Lakebox environment.

Permanently terminates and removes the specified lakebox.

Example:
  lakebox delete happy-panda-1234`,
		Args:    cobra.ExactArgs(1),
		PreRunE: mustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api := newLakeboxAPI(w)
			stderr := cmd.ErrOrStderr()

			lakeboxID := args[0]
			s := spin(stderr, fmt.Sprintf("Removing %s…", lakeboxID))

			if err := api.delete(ctx, lakeboxID); err != nil {
				s.fail(fmt.Sprintf("Failed to delete %s", lakeboxID))
				return fmt.Errorf("failed to delete lakebox %s: %w", lakeboxID, err)
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			if getDefault(profile) == lakeboxID {
				_ = clearDefault(profile)
				s.ok(fmt.Sprintf("Removed %s %s", bold(lakeboxID), dim("(default cleared)")))
			} else {
				s.ok(fmt.Sprintf("Removed %s", bold(lakeboxID)))
			}
			return nil
		},
	}

	return cmd
}
