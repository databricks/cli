package auth

import (
	"context"

	authdocker "github.com/databricks/cli/cmd/auth/docker"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func newDockerCommand() *cobra.Command {
	return authdocker.New(loadDockerToken)
}

func loadDockerToken(ctx context.Context, req authdocker.TokenRequest) (*oauth2.Token, error) {
	tokenStore, mode, err := storage.ResolveStore(ctx, storage.StorageModeUnknown)
	if err != nil {
		return nil, err
	}
	return loadToken(ctx, loadTokenArgs{
		authArguments: &auth.AuthArguments{
			Host:        req.Profile.Host,
			AccountID:   req.Profile.AccountID,
			WorkspaceID: req.Profile.WorkspaceID,
		},
		profileName:  req.Profile.Name,
		tokenTimeout: req.Timeout,
		forceRefresh: req.ForceRefresh,
		profiler:     profile.DefaultProfiler,
		tokenStore:   tokenStore,
		mode:         mode,
	})
}
