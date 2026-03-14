package auth

import (
	"context"
	"errors"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/credentials"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth/authconv"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
)

// The credentials chain used by the CLI. It is a custom implementation
// that differs from the SDK's default credentials chain. This guarantees
// that the CLI remain stable despite the evolution of the SDK while
// allowing the customization of some strategies such as "databricks-cli"
// which has a different behavior than the SDK.
//
// Modifying this order could break authentication for users whose
// environments are compatible with multiple strategies and who rely
// on the current priority for tie-breaking.
var credentialChain = []config.CredentialsStrategy{
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
}

func init() {
	// Sets the credentials chain for the CLI.
	config.DefaultCredentialStrategyProvider = func() config.CredentialsStrategy {
		return &defaultCredentials{chain: config.NewCredentialsChain(credentialChain...)}
	}
}

// defaultCredentials wraps the CLI credential chain and provides "default"
// as the fallback name, matching the SDK's DefaultCredentials behavior.
type defaultCredentials struct {
	chain config.CredentialsStrategy
}

func (d *defaultCredentials) Name() string {
	if name := d.chain.Name(); name != "" {
		return name
	}
	return "default"
}

func (d *defaultCredentials) Configure(ctx context.Context, cfg *config.Config) (credentials.CredentialsProvider, error) {
	return d.chain.Configure(ctx, cfg)
}

// CLICredentials is a credentials strategy that reads OAuth tokens directly
// from the local token store. It replaces the SDK's default "databricks-cli"
// strategy, which shells out to `databricks auth token` as a subprocess.
type CLICredentials struct {
	// persistentAuth is a function to override the default implementation
	// of the persistent auth client. It exists for testing purposes only.
	persistentAuthFn func(ctx context.Context, opts ...u2m.PersistentAuthOption) (auth.TokenSource, error)
}

// Name implements [config.CredentialsStrategy].
func (c CLICredentials) Name() string {
	return "databricks-cli"
}

var errNoHost = errors.New("no host provided")

// Configure implements [config.CredentialsStrategy].
//
// IMPORTANT: This credentials strategy ignores the scopes specified in the
// config and purely relies on the scopes from the loaded CLI token. This can
// lead to mismatches if the token was obtained with different scopes than the
// ones configured in the current profile. This is a temporary limitation that
// will be addressed in a future release by adding support for dynamic token
// downscoping.
func (c CLICredentials) Configure(ctx context.Context, cfg *config.Config) (credentials.CredentialsProvider, error) {
	if cfg.Host == "" {
		return nil, errNoHost
	}
	oauthArg, err := authArgumentsFromConfig(cfg).ToOAuthArgument()
	if err != nil {
		return nil, err
	}
	ts, err := c.persistentAuth(ctx, u2m.WithOAuthArgument(oauthArg))
	if err != nil {
		return nil, err
	}
	cp := credentials.NewOAuthCredentialsProviderFromTokenSource(
		auth.NewCachedTokenSource(ts, auth.WithAsyncRefresh(!cfg.DisableOAuthRefreshToken)),
	)
	return cp, nil
}

// persistentAuth returns a token source. It is a convenience function that
// overrides the default implementation of the persistent auth client if
// an alternative implementation is provided for testing.
func (c CLICredentials) persistentAuth(ctx context.Context, opts ...u2m.PersistentAuthOption) (auth.TokenSource, error) {
	if c.persistentAuthFn != nil {
		return c.persistentAuthFn(ctx, opts...)
	}
	ts, err := u2m.NewPersistentAuth(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return authconv.AuthTokenSource(ts), nil
}

// authArgumentsFromConfig converts an SDK config to AuthArguments.
func authArgumentsFromConfig(cfg *config.Config) AuthArguments {
	return AuthArguments{
		Host:          cfg.Host,
		AccountID:     cfg.AccountID,
		WorkspaceID:   cfg.WorkspaceID,
		IsUnifiedHost: cfg.Experimental_IsUnifiedHost,
		Profile:       cfg.Profile,
	}
}
