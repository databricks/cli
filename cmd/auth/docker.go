package auth

import (
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/spf13/cobra"
)

func newDockerCommand(authArguments *auth.AuthArguments) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "(Experimental) Manage Docker authentication for Databricks Artifact Registry",
	}
	cmd.AddCommand(newDockerTokenCommand(authArguments))
	return cmd
}

func newDockerTokenCommand(authArguments *auth.AuthArguments) *cobra.Command {
	return newDockerTokenCommandWithTokenLoader(authArguments, loadToken)
}

func newDockerTokenCommandWithTokenLoader(authArguments *auth.AuthArguments, load tokenLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "(Experimental) Generate a Docker credential",
	}

	var tokenTimeout time.Duration
	cmd.Flags().DurationVar(&tokenTimeout, "timeout", defaultTimeout, "Timeout for acquiring a token.")
	var noForceRefresh bool
	cmd.Flags().BoolVar(&noForceRefresh, "no-force-refresh", false, "Use a valid cached token instead of forcing a refresh.")

	cmd.PreRunE = validateDockerTokenRequest
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		tokenStore, mode, err := storage.ResolveStore(ctx, "")
		if err != nil {
			return err
		}
		return runDockerToken(ctx, cmd, loadTokenArgs{
			authArguments: authArguments,
			tokenTimeout:  tokenTimeout,
			// Docker may reuse one credential for a long upload, so maximize its lifetime by refreshing it by default.
			forceRefresh: !noForceRefresh,
			profiler:     profile.DefaultProfiler,
			tokenStore:   tokenStore,
			mode:         mode,
		}, load)
	}
	return cmd
}
