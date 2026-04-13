package lakebox

import (
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func newRegisterKeyCommand() *cobra.Command {
	var publicKeyFile string

	cmd := &cobra.Command{
		Use:   "register-key",
		Short: "Register an SSH public key for lakebox access",
		Long: `Register an SSH public key with the lakebox service.

Once registered, the key can be used to SSH into any of your lakeboxes.
A user can have multiple registered keys; any of them grants access to
all lakeboxes owned by that user.

Example:
  databricks lakebox register-key --public-key-file ~/.ssh/id_ed25519.pub`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api := newLakeboxAPI(w)

			if publicKeyFile == "" {
				return fmt.Errorf("--public-key-file is required")
			}

			data, err := os.ReadFile(publicKeyFile)
			if err != nil {
				return fmt.Errorf("failed to read public key file %s: %w", publicKeyFile, err)
			}

			publicKey := string(data)
			if err := api.registerKey(ctx, publicKey); err != nil {
				return fmt.Errorf("failed to register key: %w", err)
			}

			fmt.Fprintln(cmd.ErrOrStderr(), "SSH public key registered.")
			return nil
		},
	}

	cmd.Flags().StringVar(&publicKeyFile, "public-key-file", "", "Path to SSH public key file to register")
	_ = cmd.MarkFlagRequired("public-key-file")

	return cmd
}
