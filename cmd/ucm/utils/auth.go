package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/databricks/cli/cmd/root"
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
// workspaceClientOrPrompt synthesizes this error when it detects a
// wrong host type via cfg.HostType().
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
		// Only synthesize errNotWorkspaceClient for wrong host type so that
		// callers can fall through to account client.
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

// MustWorkspaceClient resolves a workspace client (with optional ucm.yml
// auth fields layered on top of profile/env), stores it on the command
// context, and returns. Used as a Cobra (Persistent)PreRunE hook on ucm
// verbs that need a live SDK client.
//
// TODO(#99): port AccountClient resolution here. v1 ucm design (see
// cmd/ucm/CLAUDE.md "Auth model") requires both clients, but A.iii.1 ships
// workspace-only to keep the fork mechanically aligned with bundle's shape.
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
	// If the profile flag is set but the target flag is not, we should skip loading the ucm configuration.
	if !isTargetFlagSet && hasProfileFlag {
		cmd.SetContext(SkipLoadUcm(cmd.Context()))
	}

	ctx = cmdctx.SetConfigUsed(cmd.Context(), cfg)
	cmd.SetContext(ctx)

	// Try to load a ucm configuration if we're allowed to by the caller.
	if !shouldSkipLoadUcm(cmd.Context()) {
		u := TryConfigureUcm(cmd)
		// Use the updated context from the command after TryConfigureUcm
		ctx = cmd.Context()
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		if u != nil {
			ctx = cmdctx.SetConfigUsed(ctx, u.Config.Workspace.Config())
			cmd.SetContext(ctx)
			client, err := u.WorkspaceClientE()
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
