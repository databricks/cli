package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/credentials"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth/authconv"
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
// IMPORTANT: This credentials strategy does not request the scopes specified
// in the config. It uses the cached CLI token, which carries the scopes from
// login. For named profiles, PersistentAuth rejects the token when the
// configured scopes differ from the ones used at login. Host-keyed tokens are
// not checked (see PersistentAuth.loadEntry), so a scope mismatch there goes
// unnoticed. Support for dynamic token downscoping in a future release would
// remove this limitation.
func (c CLICredentials) Configure(ctx context.Context, cfg *config.Config) (credentials.CredentialsProvider, error) {
	if cfg.Host == "" {
		return nil, errNoHost
	}
	oauthArg, err := authArgumentsFromConfig(cfg).ToOAuthArgument()
	if err != nil {
		return nil, err
	}
	// Without WithTokenStore, u2m.NewPersistentAuth falls back to an in-memory
	// store. Every workspace client built through this strategy would then miss
	// the token persisted by login and fail with "cache: token not found".
	tokenStore, mode, err := storage.ResolveStore(ctx, "")
	if err != nil {
		return nil, err
	}
	params, err := c.tokenParamsFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	ts, err := c.persistentAuth(ctx,
		u2m.WithOAuthArgument(oauthArg),
		u2m.WithTokenStore(storage.OAuthTokenStore(ctx, tokenStore, mode)),
		u2m.WithTokenRequestParams(params),
	)
	if err != nil {
		return nil, err
	}
	cp := credentials.NewOAuthCredentialsProviderFromTokenSource(
		auth.NewCachedTokenSource(ts, auth.WithAsyncRefresh(!cfg.DisableOAuthRefreshToken)),
	)
	return cp, nil
}

// tokenParamsFromConfig returns the login settings in cfg. Refresh does not
// send them, but PersistentAuth compares them with the ones used at login.
func (c CLICredentials) tokenParamsFromConfig(cfg *config.Config) (u2m.TokenRequestParams, error) {
	params := u2m.TokenRequestParams{Scopes: cfg.Scopes}
	// Client ID and resources belong to databricks-cli profiles only.
	// u2m.TokenRequestParamsFromProfile applies the same rule for `auth token`,
	// so both read a profile the same way.
	if cfg.AuthType != c.Name() {
		return params, nil
	}
	params.ClientID = cfg.ClientID
	// Without a profile the token is host-keyed and not checked (see
	// PersistentAuth.loadEntry), so there are no resources to read.
	if cfg.Profile == "" {
		return params, nil
	}
	resources, err := profileResources(cfg)
	if err != nil {
		return u2m.TokenRequestParams{}, err
	}
	params.Resources = resources
	return params, nil
}

// profileResources reads the profile's resources from its config file,
// because the resolved Config has no resources field. Other settings are not
// read again, so if the file is edited concurrently, resources may be newer
// than the rest of cfg.
func profileResources(cfg *config.Config) ([]string, error) {
	path := cfg.ConfigFile
	// A profile can set config_file after its file was opened. Its recorded
	// source identifies the file actually loaded, rather than that new value.
	if attr := cfg.GetAuthDetails().Configuration["config_file"]; attr != nil && attr.Source.Type == config.SourceFile {
		path = attr.Source.Name
	}
	file, err := config.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load resources for profile %q: %w", cfg.Profile, err)
	}
	section := file.Section(cfg.Profile)
	if len(section.Keys()) == 0 {
		return nil, fmt.Errorf("profile %q not found in %s", cfg.Profile, file.Path())
	}
	// Match the profile loader's own-section keys, without INI parent inheritance.
	resources := section.KeysHash()["resources"]
	if resources == "" {
		return nil, nil
	}
	// PersistentAuth trims and drops blank entries before comparing.
	return strings.Split(resources, ","), nil
}

// persistentAuth returns a token source. It is a convenience function that
// overrides the default implementation of the persistent auth client if
// an alternative implementation is provided for testing. The caller is
// responsible for supplying the token store via u2m.WithTokenStore; Configure
// does this via storage.ResolveStore so login, refresh, and all workspace
// clients share the same backend.
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
// DiscoveryURL is the primary (and only) unified-host signal; it is populated
// by EnsureResolved() from .well-known/databricks-config before this runs.
func authArgumentsFromConfig(cfg *config.Config) AuthArguments {
	return AuthArguments{
		Host:         cfg.Host,
		AccountID:    cfg.AccountID,
		WorkspaceID:  cfg.WorkspaceID,
		Profile:      cfg.Profile,
		DiscoveryURL: cfg.DiscoveryURL,
	}
}
