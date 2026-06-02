package lakebox

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// listRow embeds sandboxEntry so JSON output stays byte-identical to the
// raw API response. Text-mode fields are tagged `json:"-"` so they don't
// leak when the user passes `-o json`.
type listRow struct {
	sandboxEntry
	DisplayName string `json:"-"`
	AutoStop    string `json:"-"`
	Default     string `json:"-"`
}

// State colors are picked from cmdio's RenderFuncMap palette. green,
// yellow, and blue all emit same-byte-width SGR sequences, so the STATUS
// column stays aligned under tabwriter even when colors vary per row.
const (
	listHeaderTemplate = `{{header "ID"}}	{{header "NAME"}}	{{header "STATUS"}}	{{header "AUTOSTOP"}}	{{header "DEFAULT"}}`
	listRowTemplate    = `{{range .}}{{.SandboxID | bold}}	{{.DisplayName}}	{{if eq .Status "Running"}}{{green "%s" .Status}}{{else if eq .Status "Creating"}}{{yellow "%s" .Status}}{{else}}{{blue "%s" .Status}}{{end}}	{{.AutoStop | faint}}	{{.Default | cyan}}
{{end}}`
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your Lakebox environments",
		Long: `List your Lakebox environments.

Shows all lakeboxes associated with your account, including their
current status and ID.

Example:
  databricks lakebox list
  databricks lakebox list -o json`,
		PreRunE: root.MustWorkspaceClient,
		Annotations: map[string]string{
			"headerTemplate": listHeaderTemplate,
			"template":       listRowTemplate,
		},
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

			// JSON path: emit the raw entries (not the display-row wrapper)
			// so consumers see the same shape as the underlying API.
			if root.OutputType(cmd) == flags.OutputJSON {
				return cmdio.Render(ctx, entries)
			}

			if len(entries) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", cmdio.Faint(ctx, "No lakeboxes found."))
				return nil
			}

			rows := make([]listRow, len(entries))
			for i, e := range entries {
				// "-" stands in for an unset NAME (or a NAME that just
				// echoes the ID and so carries no extra information).
				// Keep it ASCII so it doesn't add an ANSI wrapper that
				// would throw off the column.
				name := e.Name
				if name == "" || name == e.SandboxID {
					name = "-"
				}
				def := ""
				if e.SandboxID == defaultID {
					def = "*"
				}
				rows[i] = listRow{
					sandboxEntry: e,
					DisplayName:  name,
					AutoStop:     e.autoStopLabel(),
					Default:      def,
				}
			}
			return cmdio.Render(ctx, rows)
		},
	}

	return cmd
}
