package aircmd

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newRegisterImageCommand() *cobra.Command {
	var (
		scope           string
		key             string
		interactiveAuth bool
		tagPolicy       string
		timeoutMinutes  int
	)

	cmd := &cobra.Command{
		Use:   "register-image IMAGE_URL",
		Args:  root.ExactArgs(1),
		Short: "Mirror a Docker image into the workspace registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("register-image")
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "", "Databricks secret scope holding registry credentials")
	cmd.Flags().StringVar(&key, "key", "", "Databricks secret key holding registry credentials")
	cmd.Flags().BoolVarP(&interactiveAuth, "interactive-authenticate", "i", false, "Prompt for registry credentials and store them as a secret")
	cmd.Flags().StringVar(&tagPolicy, "tag-policy", "auto", "Image resolution policy: auto or latest")
	cmd.Flags().IntVar(&timeoutMinutes, "timeout-minutes", 60, "Timeout to wait for the image to become available")

	return cmd
}
