package sandbox

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
		Use:     "default <sandbox-id>",
		Aliases: []string{"set-default"},
		Short:   "Set the default Sandbox for SSH",
		Long: `Set the default Sandbox that 'databricks sandbox ssh' connects to.

The default is stored locally in ~/.databricks/sandbox.json per profile.
The ID is validated against the server before being written, so a typo
or a sandbox that lives on a different workspace fails fast instead of
silently corrupting local state.

Example:
  databricks sandbox default happy-panda-1234`,
		Args:              cobra.ExactArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSandboxIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			sandboxID, err := resolveLocalID(ctx, profile, args[0])
			if err != nil {
				return err
			}
			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}
			entry, err := api.get(ctx, sandboxID)
			if err != nil {
				if errors.Is(err, apierr.ErrNotFound) {
					return fmt.Errorf("no sandbox named %q — `databricks sandbox list` shows available IDs", sandboxID)
				}
				return fmt.Errorf("failed to validate sandbox %s: %w", sandboxID, err)
			}

			if err := setDefault(ctx, profile, sandboxID); err != nil {
				return fmt.Errorf("failed to set default: %w", err)
			}
			_ = upsertSandbox(ctx, profile, entry.SandboxID, entry.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Default sandbox set to: %s\n", sandboxID)
			return nil
		},
	}
	return cmd
}
