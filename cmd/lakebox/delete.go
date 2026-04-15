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

Permanently terminates and removes the specified lakebox. Only the
creator (same auth token) can delete a lakebox.

Example:
  databricks lakebox delete happy-panda-1234`,
		Args:    cobra.ExactArgs(1),
		PreRunE: mustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api := newLakeboxAPI(w)

			lakeboxID := args[0]

			if err := api.delete(ctx, lakeboxID); err != nil {
				return fmt.Errorf("failed to delete lakebox %s: %w", lakeboxID, err)
			}

			// Clear default if we just deleted it.
			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			if getDefault(profile) == lakeboxID {
				_ = clearDefault(profile)
				fmt.Fprintf(cmd.ErrOrStderr(), "Cleared default lakebox.\n")
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Deleted lakebox %s\n", lakeboxID)
			return nil
		},
	}

	return cmd
}
