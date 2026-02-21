package auth

import (
	"context"
	"errors"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/credentials"
	sdkauth "github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth/authconv"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
)

func init() {
	// Sets the credentials chain for the CLI.
	//
	// The CLI relies on its own credentials chain to authenticate the user.
	// This guarantees that the CLI remain stable despite the evolution of
	// the SDK while allowing the customization of some strategies such as
	// "databricks-cli" which has a different behavior than the SDK.
	config.DefaultCredentialStrategyProvider = func() config.CredentialsStrategy {
		// Order in which strategies are tested. Iteration proceeds from most
		// specific to most generic, and the first strategy to return a non-nil
		// credentials provider is selected.
		//
		// Modifying this order could break authentication for users whose
		// environments are compatible with multiple strategies and who rely
		// on the current priority for tie-breaking.
		return config.NewCredentialsChain(
			config.PatCredentials{},
			config.BasicCredentials{},
			config.M2mCredentials{},
			CLICredentials{}, // custom
			config.MetadataServiceCredentials{},
			// OIDC Strategies.
			config.GitHubOIDCCredentials{},
			config.AzureDevOpsOIDCCredentials{},
			config.EnvOIDCCredentials{},
			config.FileOIDCCredentials{},
			// Azure strategies.
			config.AzureGithubOIDCCredentials{},
			config.AzureMsiCredentials{},
			config.AzureClientSecretCredentials{},
			config.AzureCliCredentials{},
			// Google strategies.
			config.GoogleCredentials{},
			config.GoogleDefaultCredentials{},
		)
	}
}

// CLICredentials is a credentials strategy that reads OAuth tokens directly
// from the local token store. It replaces the SDK's default "databricks-cli"
// strategy, which shells out to `databricks auth token` as a subprocess.
type CLICredentials struct {
	PersistentAuthOptions []u2m.PersistentAuthOption
}

// Name implements [config.CredentialsStrategy].
func (c CLICredentials) Name() string {
	return "databricks-cli"
}

// Configure implements [config.CredentialsStrategy].
func (c CLICredentials) Configure(ctx context.Context, cfg *config.Config) (credentials.CredentialsProvider, error) {
	if cfg.Host == "" {
		return nil, errors.New("no host provided")
	}
	oauthArg, err := authArgumentsFromConfig(cfg).ToOAuthArgument()
	if err != nil {
		return nil, err
	}
	opts := append(c.PersistentAuthOptions, u2m.WithOAuthArgument(oauthArg))
	persistentAuth, err := u2m.NewPersistentAuth(ctx, opts...)
	if err != nil {
		return nil, err
	}
	ts := sdkauth.NewCachedTokenSource(
		authconv.AuthTokenSource(persistentAuth),
		sdkauth.WithAsyncRefresh(!cfg.DisableOAuthRefreshToken),
	)
	cp := credentials.NewOAuthCredentialsProviderFromTokenSource(ts)
	return cp, nil
}

// authArgumentsFromConfig converts an SDK config to AuthArguments.
func authArgumentsFromConfig(cfg *config.Config) AuthArguments {
	return AuthArguments{
		Host:          cfg.Host,
		AccountID:     cfg.AccountID,
		WorkspaceID:   cfg.WorkspaceID,
		IsUnifiedHost: cfg.Experimental_IsUnifiedHost,
	}
}
