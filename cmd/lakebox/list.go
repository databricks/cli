package lakebox

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your Lakebox environments",
		Long: `List your Lakebox environments.

Shows all lakeboxes associated with your account, including their
current status and ID.

Example:
  databricks lakebox list
  databricks lakebox list --json`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			entries, err := api.list(ctx)
			if err != nil {
				return fmt.Errorf("failed to list lakeboxes: %w", err)
			}

			if outputJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}

			if len(entries) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", cmdio.Dim(ctx, "No lakeboxes found."))
				return nil
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			defaultID := getDefault(ctx, profile)

			out := cmd.OutOrStdout()

			// Compute column widths. AUTOSTOP holds short tokens like
			// `default`, `never`, `15m`, `1h30m` — 8 chars covers them.
			col := 10
			autostopCol := 8
			for _, e := range entries {
				if l := len(e.SandboxID); l > col {
					col = l
				}
				if l := len(e.autoStopLabel()); l > autostopCol {
					autostopCol = l
				}
			}
			col += 2
			autostopCol += 2

			blank(out)
			header := fmt.Sprintf("%-*s  %-10s  %-*s  %s",
				col, "ID", "STATUS", autostopCol, "AUTOSTOP", "DEFAULT")
			fmt.Fprintf(out, "  %s\n", cmdio.Dim(ctx, header))
			fmt.Fprintf(out, "  %s\n", cmdio.Dim(ctx, strings.Repeat("─", col+10+autostopCol+12)))

			for _, e := range entries {
				id := e.SandboxID
				def := ""
				if id == defaultID {
					def = cmdio.Cyan(ctx, "*")
				}
				// Pad ID manually so visible-width alignment is preserved
				// after the helpers wrap each cell with ANSI escapes.
				idPad := max(col-len(id), 0)
				st := status(ctx, e.Status)
				stPad := max(10-len(e.Status), 0)
				as := e.autoStopLabel()
				asPad := max(autostopCol-len(as), 0)
				idStr := cmdio.Bold(ctx, id)
				if strings.EqualFold(e.Status, "running") {
					idStr = cmdio.Bold(ctx, cmdio.Cyan(ctx, id))
				}
				fmt.Fprintf(out, "  %s%s  %s%s  %s%s  %s\n",
					idStr, strings.Repeat(" ", idPad),
					st, strings.Repeat(" ", stPad),
					cmdio.Dim(ctx, as), strings.Repeat(" ", asPad),
					def)
			}
			blank(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
