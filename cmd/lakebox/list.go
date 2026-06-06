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

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			// `list` returns the full set (the API client loops through every
			// page), so it's the cheapest place to keep local state coherent:
			//
			//   - If our saved default isn't in the result, the lakebox was
			//     deleted elsewhere — clear so the next `ssh` provisions fresh
			//     instead of erroring against a missing ID.
			//   - Cache the gateway hostname stamped on any returned entry so
			//     subsequent `ssh <id>` invocations don't need their own `get`.
			defaultID := getDefault(ctx, profile)
			if defaultID != "" {
				found := false
				for _, e := range entries {
					if e.SandboxID == defaultID {
						found = true
						break
					}
				}
				if !found {
					warn(ctx, fmt.Sprintf("Saved default %s no longer exists; clearing", defaultID))
					_ = clearDefault(ctx, profile)
					defaultID = ""
				}
			}
			for _, e := range entries {
				if e.GatewayHost != "" {
					_ = setGatewayHost(ctx, profile, e.GatewayHost)
					break
				}
			}

			// Replace the local (id, name) cache from this fresh list,
			// so subsequent name-based commands (`lakebox ssh my-project`,
			// etc.) resolve locally without another API call.
			refs := make([]cachedSandbox, 0, len(entries))
			for _, e := range entries {
				refs = append(refs, cachedSandbox{ID: e.SandboxID, Name: e.Name})
			}
			_ = setSandboxes(ctx, profile, refs)

			if jsonOutput(cmd, outputJSON) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}

			if len(entries) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", cmdio.Faint(ctx, "No lakeboxes found."))
				return nil
			}

			out := cmd.OutOrStdout()

			// NAME is always rendered (even with no --name set on any
			// row) so the table shape stays stable across calls;
			// unnamed sandboxes render as `-`. Widths are measured in
			// terminal cells via cmdio.Width so emoji / CJK names line
			// up correctly.
			idCol := 10
			autostopCol := 8
			nameCol := 4
			for _, e := range entries {
				if l := cmdio.Width(e.SandboxID); l > idCol {
					idCol = l
				}
				if l := cmdio.Width(e.autoStopLabel()); l > autostopCol {
					autostopCol = l
				}
				// A name equal to the id renders as `-`, so don't let
				// it expand the column.
				if e.Name != "" && e.Name != e.SandboxID {
					if l := cmdio.Width(e.Name); l > nameCol {
						nameCol = l
					}
				}
			}
			idCol += 2
			autostopCol += 2
			nameCol += 2
			const statusCol = 10
			const defaultCol = 7

			blank(out)
			header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %s",
				idCol, "ID", nameCol, "NAME", statusCol, "STATUS", autostopCol, "AUTOSTOP", "DEFAULT")
			fmt.Fprintf(out, "  %s\n", cmdio.Faint(ctx, header))

			ruleLen := idCol + nameCol + statusCol + autostopCol + defaultCol + 8
			fmt.Fprintf(out, "  %s\n", cmdio.Faint(ctx, strings.Repeat("─", ruleLen)))

			for _, e := range entries {
				id := e.SandboxID
				def := ""
				if id == defaultID {
					def = cmdio.Cyan(ctx, "*")
				}
				idStr := cmdio.Bold(ctx, id)
				if strings.EqualFold(e.Status, "running") {
					idStr = cmdio.Bold(ctx, cmdio.Cyan(ctx, id))
				}
				nm := cmdio.Faint(ctx, "-")
				if e.Name != "" && e.Name != id {
					nm = e.Name
				}
				// cmdio.PadRight measures visible width, so the ANSI escapes
				// the color helpers wrap each cell in don't break alignment.
				fmt.Fprintf(out, "  %s  %s  %s  %s  %s\n",
					cmdio.PadRight(idStr, idCol),
					cmdio.PadRight(nm, nameCol),
					cmdio.PadRight(status(ctx, e.Status), statusCol),
					cmdio.PadRight(cmdio.Faint(ctx, e.autoStopLabel()), autostopCol),
					def)
			}
			blank(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
