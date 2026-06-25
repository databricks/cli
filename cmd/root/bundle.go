package root

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	envlib "github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
)

// getTarget returns the name of the target to operate in.
func getTarget(cmd *cobra.Command) (value string) {
	target, isFlagSet := targetFlagValue(cmd)
	if isFlagSet {
		return target
	}

	// If it's not set, use the environment variable.
	target, _ = env.Target(cmd.Context())
	return target
}

func targetFlagValue(cmd *cobra.Command) (string, bool) {
	// The command line flag takes precedence.
	flag := cmd.Flag("target")
	if flag != nil {
		value := flag.Value.String()
		if value != "" {
			return value, true
		}
	}

	oldFlag := cmd.Flag("environment")
	if oldFlag != nil {
		value := oldFlag.Value.String()
		if value != "" {
			return value, true
		}
	}

	return "", false
}

func getProfile(cmd *cobra.Command) (value string) {
	// The command line flag takes precedence.
	flag := cmd.Flag("profile")
	if flag != nil {
		value = flag.Value.String()
		if value != "" {
			return value
		}
	}

	// If it's not set, use the environment variable.
	return envlib.Get(cmd.Context(), "DATABRICKS_CONFIG_PROFILE")
}

// configureProfile applies the profile flag to the bundle.
func configureProfile(cmd *cobra.Command, b *bundle.Bundle) error {
	profile := getProfile(cmd)

	// Fall back to [__settings__].default_profile only when the bundle does
	// not pin its own host. The legacy [DEFAULT] section is intentionally
	// NOT considered here: a hostless bundle silently routing to whatever
	// [DEFAULT] points at could deploy to the wrong workspace and mask a
	// missing workspace.host. Auth-only paths use the broader
	// databrickscfg.ResolveDefaultProfile, which also accepts [DEFAULT].
	if profile == "" && b.Config.Workspace.Host == "" && b.Config.Workspace.Profile == "" {
		configFilePath := envlib.Get(cmd.Context(), "DATABRICKS_CONFIG_FILE")
		profile, _ = databrickscfg.GetConfiguredDefaultProfile(cmd.Context(), configFilePath)
	}

	if profile == "" {
		return nil
	}

	return bundle.ApplyFuncContext(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.Profile = profile
	})
}

// resolveProfileAmbiguity resolves a multi-profile match by filtering to
// workspace-compatible profiles and either auto-selecting, prompting, or
// returning a guidance error.
func resolveProfileAmbiguity(cmd *cobra.Command, b *bundle.Bundle, originalErr error, names []string) (string, error) {
	ctx := cmd.Context()

	namesMatcher := profile.MatchProfileNames(names...)
	profiler := profile.GetProfiler(ctx)
	profiles, err := profiler.LoadProfiles(ctx, func(p profile.Profile) bool {
		return namesMatcher(p) && profile.MatchWorkspaceProfiles(p)
	})
	if err != nil {
		if errors.Is(err, profile.ErrNoConfiguration) {
			return "", originalErr
		}
		return "", err
	}

	switch len(profiles) {
	case 0:
		return "", originalErr
	case 1:
		// Exactly one workspace-compatible profile — auto-select.
		// This happens when multiple profiles match a host but only one
		// is workspace-compatible (the rest are account-only).
		return profiles[0].Name, nil
	}

	// Multiple workspace-compatible profiles — need interactive selection.
	_, hasProfileFlag := profileFlagValue(cmd)
	allowPrompt := !hasProfileFlag && !shouldSkipPrompt(ctx)
	if !allowPrompt || !cmdio.IsPromptSupported(ctx) {
		return "", fmt.Errorf(
			"%w\n\nMatching workspace profiles: %s\n\n"+
				"Fix (pick one):\n"+
				"  1. Set profile in databricks.yml:\n"+
				"       workspace:\n"+
				"         profile: %s\n"+
				"  2. Pass a flag:\n"+
				"       %s --profile %s\n"+
				"  3. Set env var:\n"+
				"       DATABRICKS_CONFIG_PROFILE=%s",
			originalErr,
			strings.Join(profiles.Names(), ", "),
			profiles[0].Name,
			cmd.CommandPath(),
			profiles[0].Name,
			profiles[0].Name,
		)
	}

	return profile.SelectProfile(ctx, profile.SelectConfig{
		Label:             "Multiple profiles match host " + b.Config.Workspace.Host,
		Profiles:          profiles,
		StartInSearchMode: true,
		ActiveTemplate:    `{{.Name | bold}}{{if .AccountID}} (account: {{.AccountID|faint}}){{end}}{{if .WorkspaceID}} (workspace: {{.WorkspaceID|faint}}){{end}}`,
		InactiveTemplate:  `{{.Name}}{{if .AccountID}} (account: {{.AccountID}}){{end}}{{if .WorkspaceID}} (workspace: {{.WorkspaceID}}){{end}}`,
		SelectedTemplate:  `{{ "Using profile" | faint }}: {{ .Name | bold }}`,
	})
}

// configureBundle loads the bundle configuration and configures flag values, if any.
func configureBundle(cmd *cobra.Command, b *bundle.Bundle) error {
	// Load bundle and select target.
	ctx := cmd.Context()
	var loadErr error
	if target := getTarget(cmd); target == "" {
		loadErr = phases.LoadDefaultTarget(ctx, b)
	} else {
		loadErr = phases.LoadNamedTarget(ctx, b, target)
	}

	if loadErr != nil {
		return loadErr
	}

	// Configure the workspace profile if the flag has been set.
	if err := configureProfile(cmd, b); err != nil {
		return err
	}

	// Set the auth configuration in the command context. This can be used
	// downstream to initialize a API client.
	//
	// Note that just initializing a workspace client and loading auth configuration
	// is a fast operation. It does not perform network I/O or invoke processes (for example the Azure CLI).
	client, err := b.WorkspaceClientE(ctx)
	if err != nil {
		names, isMulti := databrickscfg.AsMultipleProfiles(err)
		if !isMulti {
			return err
		}

		selected, resolveErr := resolveProfileAmbiguity(cmd, b, err, names)
		if resolveErr != nil {
			return resolveErr
		}

		b.Config.Workspace.Profile = selected
		b.ClearWorkspaceClient(ctx)
		client, err = b.WorkspaceClientE(ctx)
		if err != nil {
			return err
		}
	}

	ctx = cmdctx.SetConfigUsed(ctx, client.Config)
	cmd.SetContext(ctx)
	return nil
}

// MustConfigureBundle configures a bundle on the command context.
func MustConfigureBundle(cmd *cobra.Command) (*bundle.Bundle, error) {
	// A bundle may be configured on the context when testing.
	// If it is, return it immediately.
	b := bundle.GetOrNil(cmd.Context())
	if b != nil {
		return b, nil
	}

	b, err := bundle.MustLoad(cmd.Context())
	if err != nil {
		return b, err
	}
	if b != nil {
		if err := configureBundle(cmd, b); err != nil {
			return b, err
		}
	}
	return b, nil
}

// TryConfigureBundle configures a bundle on the command context
// if there is one, but doesn't fail if there isn't one.
func TryConfigureBundle(cmd *cobra.Command) (*bundle.Bundle, error) {
	// A bundle may be configured on the context when testing.
	// If it is, return it immediately.
	b := bundle.GetOrNil(cmd.Context())
	if b != nil {
		return b, nil
	}

	ctx := cmd.Context()
	b, err := bundle.TryLoad(ctx)
	if err != nil {
		return nil, err
	}
	// No bundle is fine in this case.
	if b == nil {
		return nil, nil
	}

	if err := configureBundle(cmd, b); err != nil {
		return b, err
	}
	return b, nil
}

// targetCompletion executes to autocomplete the argument to the target flag.
func targetCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	b, err := bundle.MustLoad(ctx)
	if err != nil || b == nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Load bundle but don't select a target (we're completing those).
	if err := phases.Load(ctx, b); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return slices.Collect(maps.Keys(b.Config.Targets)), cobra.ShellCompDirectiveDefault
}

func initTargetFlag(cmd *cobra.Command) {
	// To operate in the context of a bundle, all commands must take an "target" parameter.
	cmd.PersistentFlags().StringP("target", "t", "", "bundle target to use (if applicable)")
	cmd.RegisterFlagCompletionFunc("target", targetCompletion)
}

// DEPRECATED flag
func initEnvironmentFlag(cmd *cobra.Command) {
	// To operate in the context of a bundle, all commands must take an "environment" parameter.
	cmd.PersistentFlags().StringP("environment", "e", "", "bundle target to use (if applicable)")
	cmd.PersistentFlags().MarkDeprecated("environment", "use --target flag instead")
	cmd.RegisterFlagCompletionFunc("environment", targetCompletion)
}
