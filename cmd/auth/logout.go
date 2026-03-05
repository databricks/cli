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

This command deletes any cached OAuth tokens for the specified profile.
If --delete is specified, the profile is also removed from ~/.databrickscfg
(or the file specified by the DATABRICKS_CONFIG_FILE environment variable).

You will need to run "databricks auth login" to re-authenticate after
logging out.`,
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
			return errors.New("please specify a profile to log out of using --profile")
		}

		tokenCache, err := cache.NewFileTokenCache()
		if err != nil || tokenCache == nil {
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

	isU2MProfile := matchedProfile.AuthType == "databricks-cli"
	if isU2MProfile {
		err = clearTokenCache(ctx, *matchedProfile, args.profiler, args.tokenCache)
		if err != nil {
			return fmt.Errorf("failed to clear token cache: %w", err)
		}
	}

	if args.deleteProfile {
		err = databrickscfg.DeleteProfile(ctx, args.profileName, args.configFilePath)
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("Token cache cleared, but failed to remove profile. If this error persists, please check the state of the %s file: %v", args.configFilePath, err))
			return nil
		}
	}

	if isU2MProfile && args.deleteProfile {
		cmdio.LogString(ctx, fmt.Sprintf("Successfully logged out of and deleted profile %q.", args.profileName))
	} else if isU2MProfile && !args.deleteProfile {
		cmdio.LogString(ctx, fmt.Sprintf("Successfully logged out of profile %q. To remove the profile from the config file, use --delete.", args.profileName))
	} else if !isU2MProfile && args.deleteProfile {
		cmdio.LogString(ctx, fmt.Sprintf("Successfully deleted profile %q.", args.profileName))
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("No tokens to clear for profile %q. No changes were made. To remove the profile from the config file, use --delete.", args.profileName))
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
