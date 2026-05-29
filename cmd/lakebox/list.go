package lakebox

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/mattn/go-runewidth"
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

			// Compute column widths. AUTOSTOP holds short tokens like
			// `never`, `15m`, `1h30m` — 8 chars covers them. NAME is
			// rendered only when at least one entry sets a display name
			// different from the ID — there's no point in a column of
			// pet-names that duplicate the ID column.
			// All column widths are measured in *terminal cells*, not
			// bytes or runes — emoji and CJK glyphs render as 2 cells
			// despite being 1 rune / multi-byte, and using len() here
			// (which counts bytes) misaligns the row whenever a `--name`
			// includes wide characters. runewidth.StringWidth gives the
			// East-Asian-Width-corrected cell count.
			idCol := 10
			autostopCol := 8
			nameCol := 4
			showName := false
			for _, e := range entries {
				if l := runewidth.StringWidth(e.SandboxID); l > idCol {
					idCol = l
				}
				if l := runewidth.StringWidth(e.autoStopLabel()); l > autostopCol {
					autostopCol = l
				}
				if e.Name != "" && e.Name != e.SandboxID {
					showName = true
				}
				if l := runewidth.StringWidth(e.Name); l > nameCol {
					nameCol = l
				}
			}
			idCol += 2
			autostopCol += 2
			if showName {
				nameCol += 2
			}
			const statusCol = 10
			const defaultCol = 7

			blank(out)
			var header string
			if showName {
				header = fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %s",
					idCol, "ID", nameCol, "NAME", statusCol, "STATUS", autostopCol, "AUTOSTOP", "DEFAULT")
			} else {
				header = fmt.Sprintf("%-*s  %-*s  %-*s  %s",
					idCol, "ID", statusCol, "STATUS", autostopCol, "AUTOSTOP", "DEFAULT")
			}
			fmt.Fprintf(out, "  %s\n", cmdio.Faint(ctx, header))

			ruleLen := idCol + statusCol + autostopCol + defaultCol + 6
			if showName {
				ruleLen += nameCol + 2
			}
			fmt.Fprintf(out, "  %s\n", cmdio.Faint(ctx, strings.Repeat("─", ruleLen)))

			for _, e := range entries {
				id := e.SandboxID
				def := ""
				if id == defaultID {
					def = cmdio.Cyan(ctx, "*")
				}
				// Pad each cell manually so visible-width alignment is
				// preserved after the helpers wrap them with ANSI escapes.
				idPad := max(idCol-runewidth.StringWidth(id), 0)
				st := status(ctx, e.Status)
				stPad := max(statusCol-runewidth.StringWidth(e.Status), 0)
				as := e.autoStopLabel()
				asPad := max(autostopCol-runewidth.StringWidth(as), 0)
				idStr := cmdio.Bold(ctx, id)
				if strings.EqualFold(e.Status, "running") {
					idStr = cmdio.Bold(ctx, cmdio.Cyan(ctx, id))
				}
				if showName {
					nm := e.Name
					if nm == "" || nm == id {
						nm = "-"
					}
					nmPad := max(nameCol-runewidth.StringWidth(nm), 0)
					nmStr := nm
					if nm == "-" {
						nmStr = cmdio.Faint(ctx, "-")
					}
					fmt.Fprintf(out, "  %s%s  %s%s  %s%s  %s%s  %s\n",
						idStr, strings.Repeat(" ", idPad),
						nmStr, strings.Repeat(" ", nmPad),
						st, strings.Repeat(" ", stPad),
						cmdio.Faint(ctx, as), strings.Repeat(" ", asPad),
						def)
				} else {
					fmt.Fprintf(out, "  %s%s  %s%s  %s%s  %s\n",
						idStr, strings.Repeat(" ", idPad),
						st, strings.Repeat(" ", stPad),
						cmdio.Faint(ctx, as), strings.Repeat(" ", asPad),
						def)
				}
			}
			blank(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
