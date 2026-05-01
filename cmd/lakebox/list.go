package lakebox

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
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
  lakebox list
  lakebox list --json`,
		PreRunE: mustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api := newLakeboxAPI(w)

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
				fmt.Fprintf(cmd.ErrOrStderr(), "  %sNo lakeboxes found.%s\n", dm, rs)
				return nil
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			defaultID := getDefault(profile)

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
			fmt.Fprintf(out, "  %s%-*s  %-10s  %-*s  %s%s\n",
				dm, col, "ID", "STATUS", autostopCol, "AUTOSTOP", "DEFAULT", rs)
			fmt.Fprintf(out, "  %s%s%s\n", dm, strings.Repeat("─", col+10+autostopCol+12), rs)

			for _, e := range entries {
				id := e.SandboxID
				def := ""
				if id == defaultID {
					def = accent("*")
				}
				// Pad ID manually to avoid ANSI codes breaking alignment.
				idPad := col - len(id)
				if idPad < 0 {
					idPad = 0
				}
				st := status(e.Status)
				// Pad status to 10 visible chars.
				stPad := 10 - len(e.Status)
				if stPad < 0 {
					stPad = 0
				}
				as := e.autoStopLabel()
				asPad := autostopCol - len(as)
				if asPad < 0 {
					asPad = 0
				}
				idStr := bold(id)
				if strings.EqualFold(e.Status, "running") {
					idStr = cyan + bo + id + rs
				}
				fmt.Fprintf(out, "  %s%s  %s%s  %s%s  %s\n",
					idStr, strings.Repeat(" ", idPad),
					st, strings.Repeat(" ", stPad),
					dim(as), strings.Repeat(" ", asPad),
					def)
			}
			blank(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
