package lakebox

import (
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	var publicKeyFile string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Lakebox environment",
		Long: `Create a new Lakebox environment.

Creates a new personal development environment backed by a microVM.
Blocks until the lakebox is running and prints the lakebox ID.

If --public-key-file is provided, the key is installed in the lakebox's
authorized_keys so you can SSH directly. Otherwise the gateway key is used.

Example:
  databricks lakebox create
  databricks lakebox create --public-key-file ~/.ssh/id_ed25519.pub`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api := newLakeboxAPI(w)

			var publicKey string
			if publicKeyFile != "" {
				data, err := os.ReadFile(publicKeyFile)
				if err != nil {
					return fmt.Errorf("failed to read public key file %s: %w", publicKeyFile, err)
				}
				publicKey = string(data)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Creating lakebox...\n")

			result, err := api.create(ctx, publicKey)
			if err != nil {
				return fmt.Errorf("failed to create lakebox: %w", err)
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			// Set as default if no default exists, or the current default
			// has been deleted (no longer in the list).
			currentDefault := getDefault(profile)
			shouldSetDefault := currentDefault == ""
			if !shouldSetDefault && currentDefault != "" {
				// Check if the current default still exists.
				if _, err := api.get(ctx, currentDefault); err != nil {
					shouldSetDefault = true
				}
			}
			if shouldSetDefault {
				if err := setDefault(profile, result.LakeboxID); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to save default: %v\n", err)
				} else {
					fmt.Fprintf(cmd.ErrOrStderr(), "Set as default lakebox.\n")
				}
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Lakebox created (status: %s)\n", result.Status)
			fmt.Fprintln(cmd.OutOrStdout(), result.LakeboxID)
			return nil
		},
	}

	cmd.Flags().StringVar(&publicKeyFile, "public-key-file", "", "Path to SSH public key file to install in the lakebox")

	return cmd
}
