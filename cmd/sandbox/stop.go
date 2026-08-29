package sandbox

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <sandbox-id>",
		Short: "Stop a running Sandbox environment",
		Long: `Stop a running Sandbox environment.

Terminates the backing microVM but preserves the sandbox record and its
persistent storage. To restart, run 'databricks sandbox ssh' — the
gateway auto-starts a stopped sandbox on connection.

Stopping an already-stopped sandbox is a no-op.

Example:
  databricks sandbox stop happy-panda-1234`,
		Args:              cobra.ExactArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSandboxIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			sandboxID, err := resolveLocalID(ctx, profile, args[0])
			if err != nil {
				return err
			}

			s := spin(ctx, "Stopping "+sandboxID+"…")
			defer s.Close()

			updated, err := api.stop(ctx, sandboxID)
			if err != nil {
				s.fail("Failed to stop " + sandboxID)
				return fmt.Errorf("failed to stop sandbox %s: %w", sandboxID, err)
			}

			_ = setGatewayHost(ctx, profile, updated.GatewayHost)
			_ = upsertSandbox(ctx, profile, updated.SandboxID, updated.Name)

			s.ok("Stopped " + cmdio.Bold(ctx, updated.SandboxID))
			return nil
		},
	}

	return cmd
}
