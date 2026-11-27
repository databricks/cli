package sandbox

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
		Short: "List your Sandbox environments",
		Long: `List your Sandbox environments.

Shows all sandboxes associated with your account, including their
current status and ID.

Example:
  databricks sandbox list
  databricks sandbox list --json`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}

			entries, err := api.list(ctx)
			if err != nil {
				return fmt.Errorf("failed to list sandboxes: %w", err)
			}

			profile := cacheProfile(w)

			// A complete listing lets us clear defaults deleted by another client.

			defaultID := getDefault(ctx, profile)
			if defaultID != "" {
				found := false
				for i := range entries {
					e := &entries[i]
					if sandboxResourceID(e) == defaultID {
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

			// Replace the local (id, name) cache from this fresh list,
			// so subsequent name-based commands (`sandbox ssh my-project`,
			// etc.) resolve locally without another API call.
			refs := make([]cachedSandbox, 0, len(entries))
			for i := range entries {
				e := &entries[i]
				refs = append(refs, cachedSandbox{ID: sandboxResourceID(e), Name: sandboxLabel(e)})
			}
			_ = setSandboxes(ctx, profile, refs)

			if jsonOutput(cmd, outputJSON) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				var output []*sandboxJSONOutput
				for i := range entries {
					output = append(output, sandboxJSON(&entries[i]))
				}
				return enc.Encode(output)
			}

			if len(entries) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", cmdio.Faint(ctx, "No sandboxes found."))
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
			for i := range entries {
				e := &entries[i]
				if l := cmdio.Width(sandboxResourceID(e)); l > idCol {
					idCol = l
				}
				if l := cmdio.Width(autoStopLabel(e)); l > autostopCol {
					autostopCol = l
				}
				// A name equal to the id renders as `-`, so don't let
				// it expand the column.
				if sandboxLabel(e) != "" && sandboxLabel(e) != sandboxResourceID(e) {
					if l := cmdio.Width(sandboxLabel(e)); l > nameCol {
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

			for i := range entries {
				e := &entries[i]
				id := sandboxResourceID(e)
				def := ""
				if id == defaultID {
					def = cmdio.Cyan(ctx, "*")
				}
				idStr := cmdio.Bold(ctx, id)
				if strings.EqualFold(sandboxStatus(e), "running") {
					idStr = cmdio.Bold(ctx, cmdio.Cyan(ctx, id))
				}
				nm := cmdio.Faint(ctx, "-")
				if sandboxLabel(e) != "" && sandboxLabel(e) != id {
					nm = sandboxLabel(e)
				}
				// cmdio.PadRight measures visible width, so the ANSI escapes
				// the color helpers wrap each cell in don't break alignment.
				fmt.Fprintf(out, "  %s  %s  %s  %s  %s\n",
					cmdio.PadRight(idStr, idCol),
					cmdio.PadRight(nm, nameCol),
					cmdio.PadRight(status(ctx, sandboxStatus(e)), statusCol),
					cmdio.PadRight(cmdio.Faint(ctx, autoStopLabel(e)), autostopCol),
					def)
			}
			blank(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
