package docker

import (
	"time"

	"github.com/spf13/cobra"
)

const defaultTokenTimeout = time.Hour

// New returns the Docker authentication command.
func New(load TokenLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "(Experimental) Manage Docker authentication for Databricks Artifact Registry",
	}
	cmd.AddCommand(newDockerTokenCommand(load))
	cmd.AddCommand(newDockerConfigureCommand())
	cmd.AddCommand(newDockerHostCommand())
	return cmd
}
