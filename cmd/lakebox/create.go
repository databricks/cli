package lakebox

import (
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
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

Example:
  databricks lakebox create`,
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

			s := spin(ctx, "Provisioning your lakebox…")

			result, err := api.create(ctx, publicKey)
			if err != nil {
				s.fail("Failed to create lakebox")
				return fmt.Errorf("failed to create lakebox: %w", err)
			}

			s.ok("Lakebox " + cmdio.Cyan(ctx, result.SandboxID) + " is " + status(ctx, result.Status))

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			currentDefault := getDefault(ctx, profile)
			shouldSetDefault := currentDefault == ""
			if !shouldSetDefault && currentDefault != "" {
				if _, err := api.get(ctx, currentDefault); err != nil {
					shouldSetDefault = true
				}
			}
			if shouldSetDefault {
				if err := setDefault(ctx, profile, result.SandboxID); err != nil {
					warn(ctx, fmt.Sprintf("Could not save default: %v", err))
				} else {
					field(ctx, cmd.ErrOrStderr(), "default", result.SandboxID)
				}
			}

			blank(cmd.ErrOrStderr())
			fmt.Fprintln(cmd.OutOrStdout(), result.SandboxID)
			return nil
		},
	}

	cmd.Flags().StringVar(&publicKeyFile, "public-key-file", "", "Path to SSH public key file to install in the lakebox")

	return cmd
}
