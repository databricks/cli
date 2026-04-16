package root

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	envlib "github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

// errNotWorkspaceClient is a CLI-internal sentinel error. It signals that the
// configured host is an account host, not a workspace host.
//
// workspaceClientOrPrompt synthesizes this error (line ~214) when it detects a
// wrong host type via cfg.HostType(). MustAnyClient checks for it to decide
// whether to fall through and try an account client instead.
//
// The SDK exported this as databricks.ErrNotWorkspaceClient until v0.126.0. The
// SDK stopped *returning* it in v0.125.0 (host-type validation moved to host
// metadata resolution), but the CLI was already synthesizing it locally. The
// SDK removed the variable entirely in v0.127.0, so we now own it here.
var errNotWorkspaceClient = errors.New("invalid Databricks Workspace configuration - host is not a workspace host")

type ErrNoWorkspaceProfiles struct {
	path string
}

func (e ErrNoWorkspaceProfiles) Error() string {
	return e.path + " does not contain workspace profiles; please create one by running 'databricks auth login'"
}

type ErrNoAccountProfiles struct {
	path string
}

func (e ErrNoAccountProfiles) Error() string {
	return e.path + " does not contain account profiles"
}

func initProfileFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.RegisterFlagCompletionFunc("profile", profile.ProfileCompletion)
}

func profileFlagValue(cmd *cobra.Command) (string, bool) {
	profileFlag := cmd.Flag("profile")
	if profileFlag == nil {
		return "", false
	}
	value := profileFlag.Value.String()
	return value, value != ""
}

// Helper function to create an account client or prompt once if the given configuration is not valid.
func accountClientOrPrompt(ctx context.Context, cfg *config.Config, allowPrompt bool) (*databricks.AccountClient, error) {
	a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
	if err == nil {
		err = a.Config.Authenticate(emptyHttpRequest(ctx))
	}

	// If auth succeeded and we have an account ID, trust the SDK's resolution.
	// The SDK resolves host metadata (including .well-known/databricks-config)
	// during config initialization, so a successful auth means the config is valid
	// regardless of what HostType() returns from URL pattern matching.
	if err == nil && cfg.AccountID != "" {
		return a, nil
	}

	// Determine if we should prompt for a profile based on host type.
	// The SDK no longer returns ErrNotAccountClient from NewAccountClient
	// (as of v0.125.0, host-type validation was removed in favor of host
	// metadata resolution). Use HostType() to detect the wrong host type.
	var needsPrompt bool
	switch cfg.HostType() { //nolint:staticcheck // HostType() deprecated in SDK v0.127.0; SDK moving to host-agnostic behavior.
	case config.AccountHost, config.UnifiedHost:
		// Valid host type for account client, but still need account ID.
		needsPrompt = cfg.AccountID == ""
	default:
		// WorkspaceHost or unknown: wrong type for account client.
		needsPrompt = true
	}
	if !needsPrompt && err != nil && errors.Is(err, config.ErrCannotConfigureDefault) {
		needsPrompt = true
	}

	if !needsPrompt {
		return a, err
	}

	if !allowPrompt || !cmdio.IsPromptSupported(ctx) {
		if err == nil {
			err = databricks.ErrNotAccountClient
		}
		return a, err
	}

	// Try picking a profile dynamically if the current configuration is not valid.
	profile, err := AskForAccountProfile(ctx)
	if err != nil {
		return nil, err
	}
	a, err = databricks.NewAccountClient(&databricks.Config{Profile: profile})
	if err == nil {
		err = a.Config.Authenticate(emptyHttpRequest(ctx))
		if err != nil {
			return nil, err
		}
	}
	return a, err
}

func MustAnyClient(cmd *cobra.Command, args []string) (bool, error) {
	// Try to create a workspace client
	werr := MustWorkspaceClient(cmd, args)
	if werr == nil {
		return false, nil
	}

	// If the error indicates a wrong config type (workspace host used for account client,
	// or config type mismatch detected by workspaceClientOrPrompt), fall through to try
	// account client.
	if !errors.Is(werr, errNotWorkspaceClient) && !errors.As(werr, &ErrNoWorkspaceProfiles{}) {
		return false, werr
	}

	// Otherwise, the config used is account client one, so try to create an account client
	aerr := MustAccountClient(cmd, args)
	if errors.As(aerr, &ErrNoAccountProfiles{}) {
		return false, aerr
	}

	return true, aerr
}

func MustAccountClient(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{}

	// The command-line profile flag takes precedence over DATABRICKS_CONFIG_PROFILE.
	pr, hasProfileFlag := profileFlagValue(cmd)
	if hasProfileFlag {
		cfg.Profile = pr
	}

	ctx := cmd.Context()
	ctx = cmdctx.SetConfigUsed(ctx, cfg)
	cmd.SetContext(ctx)

	profiler := profile.GetProfiler(ctx)

	resolveDefaultProfile(ctx, cfg)

	if cfg.Profile == "" {
		// account-level CLI was not really done before, so here are the assumptions:
		// 1. only admins will have account configured
		// 2. 99% of admins will have access to just one account
		// hence, we don't need to create a special "DEFAULT_ACCOUNT" profile yet
		profiles, err := profiler.LoadProfiles(cmd.Context(), profile.MatchAccountProfiles)
		if err == nil && len(profiles) == 1 {
			cfg.Profile = profiles[0].Name
		}

		// if there is no config file, we don't want to fail and instead just skip it
		if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
			return err
		}
	}

	allowPrompt := !hasProfileFlag && !shouldSkipPrompt(cmd.Context())
	a, err := accountClientOrPrompt(cmd.Context(), cfg, allowPrompt)
	if err != nil {
		return renderError(ctx, cfg, err)
	}

	ctx = cmdctx.SetAccountClient(ctx, a)
	cmd.SetContext(ctx)
	return nil
}

// Helper function to create a workspace client or prompt once if the given configuration is not valid.
func workspaceClientOrPrompt(ctx context.Context, cfg *config.Config, allowPrompt bool) (*databricks.WorkspaceClient, error) {
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err == nil {
		err = w.Config.Authenticate(emptyHttpRequest(ctx))
	}

	// If auth succeeded, trust the SDK's resolution. The SDK resolves host
	// metadata (including .well-known/databricks-config) during config
	// initialization, so a successful auth means the config is valid
	// regardless of what HostType() returns from URL pattern matching.
	if err == nil {
		return w, nil
	}

	// Determine if we should prompt for a profile. The SDK no longer returns
	// ErrNotWorkspaceClient from NewWorkspaceClient (as of v0.125.0, host-type
	// validation was removed in favor of host metadata resolution). Use
	// HostType() to detect wrong host type, and check for ErrCannotConfigureDefault.
	wrongHostType := cfg.HostType() == config.AccountHost //nolint:staticcheck // HostType() deprecated in SDK v0.127.0; SDK moving to host-agnostic behavior.
	needsPrompt := wrongHostType || errors.Is(err, config.ErrCannotConfigureDefault)

	if !needsPrompt {
		return w, err
	}

	if !allowPrompt || !cmdio.IsPromptSupported(ctx) {
		// Only synthesize ErrNotWorkspaceClient for wrong host type so that
		// callers like MustAnyClient can fall through to account client.
		// For other errors (e.g. ErrCannotConfigureDefault), return the
		// original error to preserve actionable error messages.
		if wrongHostType {
			return w, errNotWorkspaceClient
		}
		return w, err
	}

	// Try picking a profile dynamically if the current configuration is not valid.
	profile, err := AskForWorkspaceProfile(ctx)
	if err != nil {
		return nil, err
	}
	w, err = databricks.NewWorkspaceClient(&databricks.Config{Profile: profile})
	if err == nil {
		err = w.Config.Authenticate(emptyHttpRequest(ctx))
		if err != nil {
			return nil, err
		}
	}
	return w, err
}

func MustWorkspaceClient(cmd *cobra.Command, args []string) error {
	ctx := logdiag.InitContext(cmd.Context())
	cmd.SetContext(ctx)

	cfg := &config.Config{}

	// The command-line profile flag takes precedence over DATABRICKS_CONFIG_PROFILE.
	profile, hasProfileFlag := profileFlagValue(cmd)
	if hasProfileFlag {
		cfg.Profile = profile
	}

	resolveDefaultProfile(ctx, cfg)

	_, isTargetFlagSet := targetFlagValue(cmd)
	// If the profile flag is set but the target flag is not, we should skip loading the bundle configuration.
	if !isTargetFlagSet && hasProfileFlag {
		cmd.SetContext(SkipLoadBundle(cmd.Context()))
	}

	ctx = cmdctx.SetConfigUsed(cmd.Context(), cfg)
	cmd.SetContext(ctx)

	// Try to load a bundle configuration if we're allowed to by the caller (see `./auth_options.go`).
	if !shouldSkipLoadBundle(cmd.Context()) {
		b := TryConfigureBundle(cmd)
		// Use the updated context from the command after TryConfigureBundle
		ctx = cmd.Context()
		if logdiag.HasError(ctx) {
			return ErrAlreadyPrinted
		}

		if b != nil {
			ctx = cmdctx.SetConfigUsed(ctx, b.Config.Workspace.Config())
			cmd.SetContext(ctx)
			client, err := b.WorkspaceClientE()
			if err != nil {
				return err
			}
			cfg = client.Config
		}
	}

	allowPrompt := !hasProfileFlag && !shouldSkipPrompt(cmd.Context())
	w, err := workspaceClientOrPrompt(cmd.Context(), cfg, allowPrompt)
	if err != nil {
		return renderError(ctx, cfg, err)
	}

	ctx = cmdctx.SetWorkspaceClient(ctx, w)
	cmd.SetContext(ctx)
	return nil
}

// resolveDefaultProfile applies the [__settings__].default_profile setting
// when no profile is specified via --profile flag or DATABRICKS_CONFIG_PROFILE.
func resolveDefaultProfile(ctx context.Context, cfg *config.Config) {
	if cfg.Profile != "" || envlib.Get(ctx, "DATABRICKS_CONFIG_PROFILE") != "" {
		return
	}
	configFilePath := envlib.Get(ctx, "DATABRICKS_CONFIG_FILE")
	resolvedProfile, err := databrickscfg.GetConfiguredDefaultProfile(ctx, configFilePath)
	if err != nil {
		log.Warnf(ctx, "Failed to load default profile: %v", err)
		return
	}
	if resolvedProfile != "" {
		cfg.Profile = resolvedProfile
	}
}

func AskForWorkspaceProfile(ctx context.Context) (string, error) {
	profiler := profile.GetProfiler(ctx)
	path, err := profiler.GetPath(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot determine Databricks config file path: %w", err)
	}
	profiles, err := profiler.LoadProfiles(ctx, profile.MatchWorkspaceProfiles)
	if err != nil {
		return "", err
	}
	switch len(profiles) {
	case 0:
		return "", ErrNoWorkspaceProfiles{path: path}
	case 1:
		return profiles[0].Name, nil
	}
	return profile.SelectProfile(ctx, profile.SelectConfig{
		Label:             "Workspace profiles defined in " + path,
		Profiles:          profiles,
		StartInSearchMode: true,
		ActiveTemplate:    `{{.Name | bold}} ({{.Host|faint}})`,
		InactiveTemplate:  `{{.Name}}`,
		SelectedTemplate:  `{{ "Using workspace profile" | faint }}: {{ .Name | bold }}`,
	})
}

func AskForAccountProfile(ctx context.Context) (string, error) {
	profiler := profile.GetProfiler(ctx)
	path, err := profiler.GetPath(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot determine Databricks config file path: %w", err)
	}
	profiles, err := profiler.LoadProfiles(ctx, profile.MatchAccountProfiles)
	if err != nil {
		return "", err
	}
	switch len(profiles) {
	case 0:
		return "", ErrNoAccountProfiles{path}
	case 1:
		return profiles[0].Name, nil
	}
	return profile.SelectProfile(ctx, profile.SelectConfig{
		Label:             "Account profiles defined in " + path,
		Profiles:          profiles,
		StartInSearchMode: true,
		ActiveTemplate:    `{{.Name | bold}} ({{.AccountID|faint}} {{.Cloud|faint}})`,
		InactiveTemplate:  `{{.Name}}`,
		SelectedTemplate:  `{{ "Using account profile" | faint }}: {{ .Name | bold }}`,
	})
}

// To verify that a client is configured correctly, we pass an empty HTTP request
// to a client's `config.Authenticate` function. Note: this functionality
// should be supported by the SDK itself.
func emptyHttpRequest(ctx context.Context) *http.Request {
	req, err := http.NewRequestWithContext(ctx, "", "", nil)
	if err != nil {
		panic(err)
	}
	return req
}

func renderError(ctx context.Context, cfg *config.Config, err error) error {
	if rewritten, newErr := auth.RewriteAuthError(ctx, cfg.Host, cfg.AccountID, cfg.Profile, err); rewritten {
		return newErr
	}
	return err
}
