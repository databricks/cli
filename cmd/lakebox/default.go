package lakebox

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func newSetDefaultCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-default <lakebox-id>",
		Short: "Set the default Lakebox for SSH",
		Long: `Set the default Lakebox that 'databricks lakebox ssh' connects to.

The default is stored locally in ~/.databricks/lakebox.json per profile.

Example:
  databricks lakebox set-default happy-panda-1234`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmdctx.WorkspaceClient(cmd.Context())
			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			lakeboxID := args[0]
			if err := setDefault(profile, lakeboxID); err != nil {
				return fmt.Errorf("failed to set default: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Default lakebox set to: %s\n", lakeboxID)
			return nil
		},
	}
	return cmd
}
