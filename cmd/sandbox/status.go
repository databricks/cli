package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "status <sandbox-id>",
		Short: "Show Sandbox environment status",
		Long: `Show detailed status of a Sandbox environment.

Example:
  databricks sandbox status happy-panda-1234
  databricks sandbox status happy-panda-1234 --json`,
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

			entry, err := api.get(ctx, sandboxID)
			if err != nil {
				if errors.Is(err, apierr.ErrNotFound) {
					return fmt.Errorf("no sandbox named %q — `databricks sandbox list` shows available IDs", sandboxID)
				}
				return fmt.Errorf("failed to get sandbox %s: %w", sandboxID, err)
			}

			_ = upsertSandbox(ctx, profile, sandboxResourceID(entry), sandboxLabel(entry))

			if jsonOutput(cmd, outputJSON) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(sandboxJSON(entry))
			}

			out := cmd.OutOrStdout()
			blank(out)
			field(ctx, out, "id", cmdio.Bold(ctx, sandboxResourceID(entry)))
			if sandboxLabel(entry) != "" {
				field(ctx, out, "name", sandboxLabel(entry))
			}
			field(ctx, out, "status", status(ctx, sandboxStatus(entry)))

			field(ctx, out, "autostop", cmdio.Faint(ctx, autoStopLabel(entry)))
			blank(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
