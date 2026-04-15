package lakebox

import (
	"encoding/json"
	"fmt"

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
  databricks lakebox list
  databricks lakebox list --json`,
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
				fmt.Fprintln(cmd.ErrOrStderr(), "No lakeboxes found.")
				return nil
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			defaultID := getDefault(profile)

			fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %-12s %s\n", "ID", "STATUS", "DEFAULT")
			for _, e := range entries {
				id := extractLakeboxID(e.Name)
				def := ""
				if id == defaultID {
					def = "*"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %-12s %s\n", id, e.Status, def)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
