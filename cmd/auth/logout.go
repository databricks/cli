package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/spf13/cobra"
)

func newLogoutCommand() *cobra.Command {
	defaultConfigPath := "~/.databrickscfg"
	if runtime.GOOS == "windows" {
		defaultConfigPath = "%USERPROFILE%\\.databrickscfg"
	}

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of a Databricks profile",
		Long: fmt.Sprintf(`Log out of a Databricks profile.

This command removes the specified profile from %s and deletes
any associated cached OAuth tokens.

You will need to run "databricks auth login" to re-authenticate after
logging out.`, defaultConfigPath),
	}

	var force bool
	var profileName string
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&profileName, "profile", "", "The profile to log out of")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if profileName == "" {
			if !cmdio.IsPromptSupported(ctx) {
				return errors.New("the command is being run in a non-interactive environment, please specify a profile to log out of using --profile")
			}
			return errors.New("please specify a profile to log out of using --profile")
		}

		tokenCache, err := cache.NewFileTokenCache()
		if err != nil {
			log.Warnf(ctx, "Failed to open token cache: %v", err)
		}

		return runLogout(ctx, logoutArgs{
			profileName:    profileName,
			force:          force,
			profiler:       profile.DefaultProfiler,
			tokenCache:     tokenCache,
			configFilePath: os.Getenv("DATABRICKS_CONFIG_FILE"),
		})
	}

	return cmd
}

type logoutArgs struct {
	profileName    string
	force          bool
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

		question := fmt.Sprintf(
			"WARNING: This will remove profile %q from %s and delete "+
				"any cached OAuth tokens associated with it. You will need to run "+
				"\"databricks auth login\" to re-authenticate.\n\nAre you sure?",
			args.profileName, args.configFilePath)

		approved, err := cmdio.AskYesOrNo(ctx, question)
		if err != nil {
			return err
		}
		if !approved {
			return nil
		}
	}

	clearTokenCache(ctx, *matchedProfile, args.profiler, args.tokenCache)

	err = databrickscfg.DeleteProfile(ctx, args.profileName, args.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to remove profile: %w", err)
	}

	return nil
}

// getMatchingProfile loads a profile by name and returns an error with
// available profile names if the profile is not found.
func getMatchingProfile(ctx context.Context, profileName string, profiler profile.Profiler) (*profile.Profile, error) {
	if profiler == nil {
		return nil, errors.New("profiler cannot be nil")
	}

	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(profileName))
	if err != nil {
		return nil, err
	}

	if len(profiles) == 0 {
		allProfiles, err := profiler.LoadProfiles(ctx, profile.MatchAllProfiles)
		if err != nil {
			return nil, fmt.Errorf("profile %q not found", profileName)
		}

		return nil, fmt.Errorf("profile %q not found. Available profiles: %s", profileName, allProfiles.Names())
	}

	return &profiles[0], nil
}

// clearTokenCache removes cached OAuth tokens for the given profile from the
// token cache. It removes:
//  1. The entry keyed by the profile name.
//  2. The entry keyed by the host URL, but only if no other remaining profile
//     references the same host.
func clearTokenCache(ctx context.Context, p profile.Profile, profiler profile.Profiler, tokenCache cache.TokenCache) {
	if tokenCache == nil {
		return
	}

	profileName := p.Name
	if err := tokenCache.Store(profileName, nil); err != nil {
		log.Warnf(ctx, "Failed to delete profile-keyed token for profile %q: %v", profileName, err)
	}

	host := strings.TrimRight(p.Host, "/")
	if host == "" {
		return
	}

	otherProfilesUsingHost, err := profiler.LoadProfiles(ctx, func(candidate profile.Profile) bool {
		return candidate.Name != profileName && profile.WithHost(host)(candidate)
	})
	if err != nil {
		log.Warnf(ctx, "Failed to load profiles using host %q: %v", host, err)
		return
	}

	if len(otherProfilesUsingHost) == 0 {
		if err := tokenCache.Store(host, nil); err != nil {
			log.Warnf(ctx, "Failed to delete host-keyed token for host %q: %v", host, err)
		}
	}
}
