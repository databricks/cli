package docker

import (
	"time"

	"github.com/databricks/cli/libs/databrickscfg/profile"
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

func newDockerTokenCommand(load TokenLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "(Experimental) Generate a Docker credential",
	}

	var tokenTimeout time.Duration
	cmd.Flags().DurationVar(&tokenTimeout, "timeout", defaultTokenTimeout, "Timeout for acquiring a token.")
	var noForceRefresh bool
	cmd.Flags().BoolVar(&noForceRefresh, "no-force-refresh", false, "Use a valid cached token instead of forcing a refresh.")

	cmd.PreRunE = validateDockerTokenRequest
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		return runDockerToken(ctx, cmd, tokenOptions{
			tokenTimeout: tokenTimeout,
			// Docker may reuse one credential for a long upload, so maximize its lifetime by refreshing it by default.
			forceRefresh: !noForceRefresh,
			profiler:     profile.DefaultProfiler,
		}, load)
	}
	return cmd
}
