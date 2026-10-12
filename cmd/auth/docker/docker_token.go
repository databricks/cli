package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// TokenRequest contains the resolved profile and token acquisition options.
type TokenRequest struct {
	Profile      profile.Profile
	Timeout      time.Duration
	ForceRefresh bool
}

// TokenLoader acquires an OAuth token for a Docker credential request.
type TokenLoader func(context.Context, TokenRequest) (*oauth2.Token, error)

type dockerGetResponse struct {
	Username string `json:"Username"`
	Secret   string `json:"Secret"`
}

func newDockerTokenCommand(load TokenLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "token",
		Short:   "(Experimental) Generate a Docker credential",
		PreRunE: validateDockerTokenRequest,
	}
	var tokenTimeout time.Duration
	cmd.Flags().DurationVar(&tokenTimeout, "timeout", defaultTokenTimeout, "Timeout for acquiring a token.")
	var noForceRefresh bool
	cmd.Flags().BoolVar(&noForceRefresh, "no-force-refresh", false, "Use a valid cached token instead of forcing a refresh.")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		p, err := dockerTokenProfile(cmd.Context(), cmd.InOrStdin())
		if err != nil {
			return err
		}
		token, err := load(cmd.Context(), TokenRequest{
			Profile: p,
			Timeout: tokenTimeout,
			// Docker may reuse one credential for a long upload, so maximize its lifetime.
			ForceRefresh: !noForceRefresh,
		})
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(dockerGetResponse{
			Username: dockercredentials.OAuthTokenUsername,
			Secret:   token.AccessToken,
		})
	}
	return cmd
}

func dockerTokenProfile(ctx context.Context, input io.Reader) (profile.Profile, error) {
	raw, err := io.ReadAll(input)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("read Docker credential request: %w", err)
	}
	registry, err := dockercredentials.ParseRegistryHost(string(raw))
	if err != nil {
		return profile.Profile{}, err
	}
	return dockercredentials.ProfileForRegistry(ctx, profile.DefaultProfiler, registry)
}

func validateDockerTokenRequest(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errors.New("auth docker token does not accept positional arguments")
	}
	for _, name := range []string{"profile", "host", "account-id", "workspace-id"} {
		flag := cmd.Flag(name)
		if flag != nil && flag.Changed {
			return fmt.Errorf("auth docker token does not support --%s", name)
		}
	}
	return nil
}
