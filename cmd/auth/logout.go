package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		Use:    "logout",
		Short:  "Log out of a Databricks profile",
		Hidden: true,
		Long: `Log out of a Databricks profile.

This command clears any cached OAuth tokens for the specified profile so
that the next CLI invocation requires re-authentication. The profile
entry in ~/.databrickscfg is left intact unless --delete is also specified.

This command requires a profile to be specified (using --profile) or an
interactive terminal. If you omit --profile and run in an interactive
terminal, you'll be shown a profile picker. In a non-interactive
environment (e.g. CI/CD), omitting --profile is an error.

1. If you specify --profile, the command logs out of that profile. In an
   interactive terminal you'll be asked to confirm unless --force is set.

2. If you omit --profile in an interactive terminal, you'll be shown
   an interactive picker listing all profiles from your configuration file.
   You can search by profile name, host, or account ID. After selecting a
   profile, you'll be asked to confirm unless --force is specified.

3. If you omit --profile in a non-interactive environment (e.g. CI/CD pipeline),
   the command will fail with an error asking you to specify --profile.

4. Use --force to skip the confirmation prompt. This is required when
   running in non-interactive environments.

5. Use --delete to also remove the selected profile from ~/.databrickscfg.`,
	}

	var force bool
	var profileName string
	var deleteProfile bool
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&profileName, "profile", "", "The profile to log out of")
	cmd.Flags().BoolVar(&deleteProfile, "delete", false, "Delete the profile from the config file")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if profileName == "" {
			if !cmdio.IsPromptSupported(ctx) {
				return errors.New("the command is being run in a non-interactive environment, please specify a profile to log out of using --profile")
			}
			allProfiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.MatchAllProfiles)
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

		tokenCache, err := cache.NewFileTokenCache()
		if err != nil {
			return fmt.Errorf("failed to open token cache, please check if the file version is up-to-date and that the file is not corrupted: %w", err)
		}

		return runLogout(ctx, logoutArgs{
			profileName:    profileName,
			force:          force,
			deleteProfile:  deleteProfile,
			profiler:       profile.DefaultProfiler,
			tokenCache:     tokenCache,
			configFilePath: env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
		})
	}

	return cmd
}

type logoutArgs struct {
	profileName    string
	force          bool
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

	if !args.force {
		if !cmdio.IsPromptSupported(ctx) {
			return errors.New("please specify --force to skip confirmation in non-interactive mode")
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

	otherProfiles, err := profiler.LoadProfiles(ctx, func(candidate profile.Profile) bool {
		return candidate.Name != p.Name && matchFn(candidate)
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

// hostCacheKeyAndMatchFn returns the token cache key and a profile match
// function for the host-based token entry. Account and unified profiles use
// host/oidc/accounts/<account_id> as the cache key and match on both host and
// account ID; workspace profiles use just the host.
func hostCacheKeyAndMatchFn(p profile.Profile) (string, profile.ProfileMatchFunction) {
	host := (&config.Config{Host: p.Host}).CanonicalHostName()
	if host == "" {
		return "", nil
	}

	if p.AccountID != "" {
		return host + "/oidc/accounts/" + p.AccountID, profile.WithHostAndAccountID(host, p.AccountID)
	}

	return host, profile.WithHost(host)
}
