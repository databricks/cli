// The auth logout command was implemented across three stacked PRs that were
// inadvertently squashed into a single commit cb3c326 (titled after #4647 only):
//
//   - #4613: core logout command with --profile, --auto-approve (originally --force),
//     --delete flags, token cache cleanup, and DeleteProfile in libs/databrickscfg/ops.go.
//     https://github.com/databricks/cli/pull/4613
//
//   - #4616: interactive profile picker when --profile is
//     omitted in an interactive terminal.
//     https://github.com/databricks/cli/pull/4616
//
//   - #4647: extract shared SelectProfile helper, deduplicate
//     profile pickers across auth logout, auth token, cmd/root/auth.go, cmd/root/bundle.go.
//     https://github.com/databricks/cli/pull/4647

package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/spf13/cobra"
)

const logoutWarningTemplate = `{{ "Warning" | yellow }}: This will {{ if .DeleteProfile }}log out of and delete profile {{ .ProfileName | bold }}{{ else }}log out of profile {{ .ProfileName | bold }}{{ end }}.

The following changes will be made:
{{- if .DeleteProfile }}
  - Remove profile {{ .ProfileName | bold }} from {{ .ConfigPath }}
{{- end }}
  - Delete any cached OAuth tokens for this profile

You will need to run {{ "databricks auth login" | bold }} to re-authenticate.
`

func newLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout [PROFILE]",
		Short: "Log out of a Databricks profile",
		Args:  cobra.MaximumNArgs(1),
		Long: `Log out of a Databricks profile.

This command clears any cached OAuth tokens for the specified profile so
that the next CLI invocation requires re-authentication. The profile
entry in ~/.databrickscfg is left intact unless --delete is also specified.

This only affects profiles created with "databricks auth login". Profiles
using other authentication methods (personal access tokens, M2M credentials)
do not store cached OAuth tokens. If multiple profiles share the same cached
token, logging out of one does not affect the others.

You can provide a profile name as a positional argument, or use --profile
to specify it explicitly.

1. If you specify a profile (via argument or --profile), the command logs
   out of that profile. In an interactive terminal you'll be asked to
   confirm unless --auto-approve is set.

2. If you omit the profile in an interactive terminal, you'll be shown
   an interactive picker listing all profiles from your configuration file.
   You can search by profile name, host, or account ID. After selecting a
   profile, you'll be asked to confirm unless --auto-approve is specified.

3. If you omit the profile in a non-interactive environment (e.g. CI/CD
   pipeline), the command will fail with an error asking you to specify
   a profile.

4. Use --delete to also remove the selected profile from the configuration
   file.`,
	}

	var autoApprove bool
	var deleteProfile bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&deleteProfile, "delete", false, "Delete the profile from the config file")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		profiler := profile.DefaultProfiler

		profileFlag := cmd.Flag("profile")
		profileName := profileFlag.Value.String()

		// The positional argument is a shorthand that resolves to either a
		// profile or a host. It cannot be combined with explicit flags.
		if profileFlag.Changed && len(args) == 1 {
			return fmt.Errorf("argument %q cannot be combined with --profile. Use the --profile flag instead", args[0])
		}
		if len(args) == 1 {
			resolvedProfile, resolvedHost, err := resolvePositionalArg(ctx, args[0], profiler)
			if err != nil {
				return err
			}
			if resolvedProfile != "" {
				profileName = resolvedProfile
			} else {
				profileName, err = resolveHostToProfile(ctx, resolvedHost, profiler)
				if err != nil {
					return err
				}
			}
		}

		if profileName == "" {
			if !cmdio.IsPromptSupported(ctx) {
				return errors.New("the command is being run in a non-interactive environment, please specify a profile using the PROFILE argument or --profile flag")
			}
			allProfiles, err := profiler.LoadProfiles(ctx, profile.MatchAllProfiles)
			if err != nil {
				return err
			}
			selected, err := profile.SelectProfile(ctx, profile.SelectConfig{
				Label:             "Select a profile to log out of",
				Profiles:          allProfiles,
				StartInSearchMode: len(allProfiles) > 5,
				ActiveTemplate:    `▸ {{.PaddedName | bold}}{{if .AccountID}} (account: {{.AccountID}}){{else}} ({{.Host}}){{end}}`,
				InactiveTemplate:  `  {{.PaddedName}}{{if .AccountID}} (account: {{.AccountID | faint}}){{else}} ({{.Host | faint}}){{end}}`,
				SelectedTemplate:  `{{ "Selected profile" | faint }}: {{ .Name | bold }}`,
			})
			if err != nil {
				return err
			}
			profileName = selected
		}

		tokenCache, _, err := newAuthCache(ctx, "")
		if err != nil {
			return fmt.Errorf("failed to open token cache: %w", err)
		}

		return runLogout(ctx, logoutArgs{
			profileName:    profileName,
			autoApprove:    autoApprove,
			deleteProfile:  deleteProfile,
			profiler:       profiler,
			tokenCache:     tokenCache,
			configFilePath: env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
		})
	}

	return cmd
}

type logoutArgs struct {
	profileName    string
	autoApprove    bool
	deleteProfile  bool
	profiler       profile.Profiler
	tokenCache     cache.TokenCache
	configFilePath string
}

func runLogout(ctx context.Context, args logoutArgs) error {
	matchedProfile, err := getMatchingProfile(ctx, args.profileName, args.profiler)
	if err != nil {
		return err
	}

	if !args.autoApprove {
		if !cmdio.IsPromptSupported(ctx) {
			return errors.New("please specify --auto-approve to skip confirmation in non-interactive mode")
		}

		configPath, err := args.profiler.GetPath(ctx)
		if err != nil {
			return err
		}
		err = cmdio.RenderWithTemplate(ctx, map[string]any{
			"ProfileName":   args.profileName,
			"ConfigPath":    configPath,
			"DeleteProfile": args.deleteProfile,
		}, "", logoutWarningTemplate)
		if err != nil {
			return err
		}

		approved, err := cmdio.AskYesOrNo(ctx, "Are you sure?")
		if err != nil {
			return err
		}
		if !approved {
			cmdio.LogString(ctx, "Aborting logout... No changes were made.")
			return nil
		}
	}

	// First try to clear the token cache. If that fails do NOT try to delete
	// the profile even if --delete is specified. This avoids problems where
	// tokens are partially or completely present in the cache, but the profile
	// has been deleted. In this scenario, the user would not be able to
	// correctly delete the tokens in an eventual logout re-try.

	// To keep the symmetry between the login and logout commands, we only
	// want to clear the token cache if the profile was created by the login command.
	// Otherwise, we could be deleting profiles that were created by other means (e.g. manually).
	isCreatedByLogin := matchedProfile.AuthType == "databricks-cli"
	if isCreatedByLogin {
		err = clearTokenCache(ctx, *matchedProfile, args.profiler, args.tokenCache)
		if err != nil {
			return fmt.Errorf("failed to clear token cache: %w", err)
		}
	}

	if args.deleteProfile {
		err = databrickscfg.DeleteProfile(ctx, args.profileName, args.configFilePath)
		if err != nil {
			if isCreatedByLogin {
				return fmt.Errorf("token cache cleared, but failed to delete profile. Re-run with --delete to retry. If this error persists, please check the state of the config file: %w", err)
			}

			return fmt.Errorf("failed to delete profile. Re-run with --delete to retry. If this error persists, please check the state of the config file: %w", err)
		}

		// If the deleted profile was the configured default, clear the pointer.
		err = databrickscfg.ClearDefaultProfile(ctx, args.profileName, args.configFilePath)
		if err != nil {
			return fmt.Errorf("profile deleted, but failed to clear default profile setting: %w", err)
		}
	}

	if isCreatedByLogin && args.deleteProfile {
		cmdio.LogString(ctx, fmt.Sprintf("Logged out of and deleted profile %q.", args.profileName))
	} else if isCreatedByLogin && !args.deleteProfile {
		cmdio.LogString(ctx, fmt.Sprintf("Logged out of profile %q. Use --delete to also remove it from the config file.", args.profileName))
	} else if !isCreatedByLogin && args.deleteProfile {
		cmdio.LogString(ctx, fmt.Sprintf("Deleted profile %q with no tokens to clear.", args.profileName))
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("No tokens to clear for profile %q. Use --delete to remove it from the config file.", args.profileName))
	}
	return nil
}

// getMatchingProfile loads a profile by name and returns an error with
// available profile names if the profile is not found.
func getMatchingProfile(ctx context.Context, profileName string, profiler profile.Profiler) (*profile.Profile, error) {
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(profileName))
	if err != nil {
		return nil, err
	}

	if len(profiles) == 0 {
		allProfiles, err := profiler.LoadProfiles(ctx, profile.MatchAllProfiles)
		if err != nil {
			return nil, fmt.Errorf("profile %q not found", profileName)
		}

		names := strings.Join(allProfiles.Names(), ", ")
		return nil, fmt.Errorf("profile %q not found. Available profiles: %s", profileName, names)
	}

	return &profiles[0], nil
}

// clearTokenCache removes cached OAuth tokens for the given profile from the
// token cache. It removes:
//  1. The entry keyed by the profile name.
//  2. The entry keyed by the host-based cache key, but only if no other
//     remaining profile references the same key. For account and unified
//     profiles, the cache key includes the OIDC path
//     (host/oidc/accounts/<account_id>).
func clearTokenCache(ctx context.Context, p profile.Profile, profiler profile.Profiler, tokenCache cache.TokenCache) error {
	if err := tokenCache.Store(p.Name, nil); err != nil {
		return fmt.Errorf("failed to delete profile-keyed token for profile %q: %w", p.Name, err)
	}

	hostCacheKey, matchFn := hostCacheKeyAndMatchFn(p)
	if hostCacheKey == "" {
		return fmt.Errorf("failed to get host-based cache key for profile %q", p.Name)
	}

	// Only preserve the host-keyed token if another U2M profile shares the
	// same host. Non-U2M profiles (PAT, M2M, etc.) never use the OAuth
	// token cache, so they should not prevent cleanup.
	otherProfiles, err := profiler.LoadProfiles(ctx, func(candidate profile.Profile) bool {
		return candidate.Name != p.Name && candidate.AuthType == "databricks-cli" && matchFn(candidate)
	})
	if err != nil {
		return fmt.Errorf("failed to load profiles for host cache key %q: %w", hostCacheKey, err)
	}

	if len(otherProfiles) == 0 {
		if err := tokenCache.Store(hostCacheKey, nil); err != nil {
			return fmt.Errorf("failed to delete host-keyed token for %q: %w", hostCacheKey, err)
		}
	}
	return nil
}

// hostCacheKeyAndMatchFn returns the host-based token cache key and a profile
// match function for the given profile. The match function is used to check
// whether other profiles share the same host-based cache entry.
func hostCacheKeyAndMatchFn(p profile.Profile) (string, profile.ProfileMatchFunction) {
	// Use ToOAuthArgument to derive the host-based cache key via the same
	// routing logic the SDK used when the token was written during login.
	// This includes a .well-known/databricks-config call that distinguishes
	// classic workspace hosts from SPOG hosts — a distinction that cannot
	// be made from the profile fields alone.
	arg, err := (auth.AuthArguments{
		Host:          p.Host,
		AccountID:     p.AccountID,
		WorkspaceID:   p.WorkspaceID,
		IsUnifiedHost: p.IsUnifiedHost,
		// Profile is deliberately empty so GetCacheKey returns the host-based
		// key rather than the profile name.
		// DiscoveryURL is left empty to force a fresh .well-known resolution
		// so that the routing decision reflects the host's current state.
	}).ToOAuthArgument()
	if err != nil {
		return "", nil
	}
	hostCacheKey := arg.GetCacheKey()

	host := (&config.Config{Host: p.Host}).CanonicalHostName()
	if p.AccountID != "" && strings.Contains(hostCacheKey, "/oidc/accounts/") {
		return hostCacheKey, profile.WithHostAndAccountID(host, p.AccountID)
	}
	return hostCacheKey, profile.WithHost(host)
}
