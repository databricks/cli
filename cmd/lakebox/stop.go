package lakebox

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <lakebox-id>",
		Short: "Stop a running Lakebox environment",
		Long: `Stop a running Lakebox environment.

Terminates the backing microVM but preserves the sandbox record and its
persistent storage. To restart, run 'databricks lakebox ssh' — the
gateway auto-starts a stopped sandbox on connection.

Stopping an already-stopped sandbox is a no-op.

Example:
  databricks lakebox stop happy-panda-1234`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			lakeboxID := args[0]
			s := spin(ctx, "Stopping "+lakeboxID+"…")
			defer s.Close()

			updated, err := api.stop(ctx, lakeboxID)
			if err != nil {
				s.fail("Failed to stop " + lakeboxID)
				return fmt.Errorf("failed to stop lakebox %s: %w", lakeboxID, err)
			}
			s.ok("Stopped " + cmdio.Bold(ctx, updated.SandboxID))
			return nil
		},
	}

	return cmd
}
