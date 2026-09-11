package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type tokenLoader func(context.Context, loadTokenArgs) (*oauth2.Token, error)

type dockerGetResponse struct {
	Username string `json:"Username"`
	Secret   string `json:"Secret"`
}

func runDockerToken(ctx context.Context, cmd *cobra.Command, args loadTokenArgs, load tokenLoader) error {
	rawServer, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return fmt.Errorf("read Docker credential request: %w", err)
	}
	registry, err := dockercredentials.ParseRegistryHost(string(rawServer))
	if err != nil {
		return err
	}

	selectedProfile, err := dockerTokenProfile(ctx, registry, args.profiler)
	if err != nil {
		return err
	}

	args.authArguments = &auth.AuthArguments{
		Host:        selectedProfile.Host,
		AccountID:   selectedProfile.AccountID,
		WorkspaceID: selectedProfile.WorkspaceID,
	}
	args.profileName = selectedProfile.Name
	args.args = nil

	t, err := load(ctx, args)
	if err != nil {
		return err
	}

	return json.NewEncoder(cmd.OutOrStdout()).Encode(dockerGetResponse{
		Username: dockercredentials.OAuthTokenUsername,
		Secret:   t.AccessToken,
	})
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

func dockerTokenProfile(ctx context.Context, registry dockercredentials.Registry, profiler profile.Profiler) (profile.Profile, error) {
	workspaceProfiles, err := profiler.LoadProfiles(ctx, func(p profile.Profile) bool {
		return p.WorkspaceID == registry.WorkspaceID
	})
	if err != nil {
		return profile.Profile{}, err
	}
	if len(workspaceProfiles) == 0 {
		return profile.Profile{}, fmt.Errorf("no Databricks profile found for workspace ID %s from registry host %s. Run databricks auth login --host <workspace-url> and set workspace_id for that profile", registry.WorkspaceID, registry.Host)
	}

	var matchingProfiles profile.Profiles
	for _, p := range workspaceProfiles {
		if validateDockerCredentialProfile(p) == nil {
			matchingProfiles = append(matchingProfiles, p)
		}
	}
	switch len(matchingProfiles) {
	case 0:
		return profile.Profile{}, validateDockerCredentialProfile(workspaceProfiles[0])
	case 1:
		return matchingProfiles[0], nil
	default:
		return profile.Profile{}, fmt.Errorf("multiple Databricks profiles match workspace ID %s: %s. Remove duplicate workspace_id entries before using Docker credential helper", registry.WorkspaceID, strings.Join(matchingProfiles.Names(), " and "))
	}
}
