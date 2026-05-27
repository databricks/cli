package lakebox

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/spf13/cobra"
)

func newSetDefaultCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "default <lakebox-id>",
		Aliases: []string{"set-default"},
		Short:   "Set the default Lakebox for SSH",
		Long: `Set the default Lakebox that 'databricks lakebox ssh' connects to.

The default is stored locally in ~/.databricks/lakebox.json per profile.
The ID is validated against the server before being written, so a typo
or a sandbox that lives on a different workspace fails fast instead of
silently corrupting local state.

Example:
  databricks lakebox default happy-panda-1234`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			lakeboxID := args[0]
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}
			if _, err := api.get(ctx, lakeboxID); err != nil {
				if errors.Is(err, apierr.ErrNotFound) {
					return fmt.Errorf("no lakebox named %q — `databricks lakebox list` shows available IDs", lakeboxID)
				}
				return fmt.Errorf("failed to validate lakebox %s: %w", lakeboxID, err)
			}

			if err := setDefault(ctx, profile, lakeboxID); err != nil {
				return fmt.Errorf("failed to set default: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Default lakebox set to: %s\n", lakeboxID)
			return nil
		},
	}
	return cmd
}
