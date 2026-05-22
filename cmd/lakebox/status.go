package lakebox

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "status <lakebox-id>",
		Short: "Show Lakebox environment status",
		Long: `Show detailed status of a Lakebox environment.

Example:
  lakebox status happy-panda-1234
  lakebox status happy-panda-1234 --json`,
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

			entry, err := api.get(ctx, lakeboxID)
			if err != nil {
				return fmt.Errorf("failed to get lakebox %s: %w", lakeboxID, err)
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			_ = setGatewayHost(ctx, profile, entry.GatewayHost)

			if outputJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entry)
			}

			out := cmd.OutOrStdout()
			blank(out)
			field(ctx, out, "id", cmdio.Bold(ctx, entry.SandboxID))
			if entry.Name != "" {
				field(ctx, out, "name", entry.Name)
			}
			field(ctx, out, "status", status(ctx, entry.Status))
			if entry.FQDN != "" {
				field(ctx, out, "fqdn", cmdio.Dim(ctx, entry.FQDN))
			}
			field(ctx, out, "autostop", cmdio.Dim(ctx, entry.autoStopLabel()))
			blank(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
