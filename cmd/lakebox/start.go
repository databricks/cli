package lakebox

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <lakebox-id>",
		Short: "Start a stopped Lakebox environment",
		Long: `Start a stopped Lakebox environment.

Boots the backing microVM and brings the sandbox to Running.
'databricks lakebox ssh' already auto-starts a stopped sandbox on
connection, so this command is mostly useful for pre-warming an
environment without immediately connecting.

Starting an already-running sandbox is a no-op.

Example:
  databricks lakebox start happy-panda-1234`,
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

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			lakeboxID, err := resolveLocalID(ctx, profile, args[0])
			if err != nil {
				return err
			}

			s := spin(ctx, "Starting "+lakeboxID+"…")
			defer s.Close()

			updated, err := api.start(ctx, lakeboxID)
			if err != nil {
				s.fail("Failed to start " + lakeboxID)
				return fmt.Errorf("failed to start lakebox %s: %w", lakeboxID, err)
			}

			_ = setGatewayHost(ctx, profile, updated.GatewayHost)
			_ = upsertSandbox(ctx, profile, updated.SandboxID, updated.Name)

			s.ok("Started " + cmdio.Bold(ctx, updated.SandboxID))
			return nil
		},
	}

	return cmd
}
