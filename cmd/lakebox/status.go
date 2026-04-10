package lakebox

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "status <lakebox-id>",
		Short: "Show Lakebox environment status",
		Long: `Show detailed status of a Lakebox environment.

Example:
  databricks lakebox status happy-panda-1234
  databricks lakebox status happy-panda-1234 --json`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api := newLakeboxAPI(w)

			lakeboxID := args[0]

			entry, err := api.get(ctx, lakeboxID)
			if err != nil {
				return fmt.Errorf("failed to get lakebox %s: %w", lakeboxID, err)
			}

			if outputJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entry)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "ID:     %s\n", extractLakeboxID(entry.Name))
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", entry.Status)
			if entry.FQDN != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "FQDN:   %s\n", entry.FQDN)
			}
			if entry.PubkeyHashPrefix != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Key:    %s\n", entry.PubkeyHashPrefix)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
