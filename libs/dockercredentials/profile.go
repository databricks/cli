package dockercredentials

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
)

// ValidateProfile verifies that p can be used by the Docker credential helper.
func ValidateProfile(p profile.Profile) error {
	if p.HasClientCredentials {
		return fmt.Errorf("profile %q uses client credentials. Docker credential helper requires a profile created by databricks auth login", p.Name)
	}
	if p.AuthType != auth.AuthTypeDatabricksCli {
		return fmt.Errorf("profile %q uses auth_type %q. Docker credential helper requires a profile created by databricks auth login", p.Name, p.AuthType)
	}
	if isAccountOnlyProfile(p) {
		return fmt.Errorf("profile %q does not target a workspace. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name)
	}
	return nil
}

func isAccountOnlyProfile(p profile.Profile) bool {
	if p.Host == "" {
		return true
	}
	cfg := &config.Config{Host: p.Host, AccountID: p.AccountID, WorkspaceID: p.WorkspaceID}
	if auth.IsClassicAccountHost(cfg.CanonicalHostName()) {
		return true
	}
	return p.AccountID != "" && (p.WorkspaceID == "" || p.WorkspaceID == profile.WorkspaceIDNone)
}

// LoadProfile loads and validates the named Docker credential helper profile.
func LoadProfile(ctx context.Context, profiler profile.Profiler, name string) (profile.Profile, error) {
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(name))
	if err != nil {
		return profile.Profile{}, err
	}
	if len(profiles) == 0 {
		return profile.Profile{}, fmt.Errorf("profile %q not found", name)
	}
	if err := ValidateProfile(profiles[0]); err != nil {
		return profile.Profile{}, err
	}
	return profiles[0], nil
}

// RecordProfile marks the named profile as the Docker credential helper profile for registry and returns the names of the profiles it unmarked.
func RecordProfile(ctx context.Context, registry Registry, name string) ([]string, error) {
	// Unsupported profiles are included so a stale mark is cleared even while the profile is unusable.
	recorded, err := profile.DefaultProfiler.LoadProfiles(ctx, func(p profile.Profile) bool {
		return p.DockerCredentialHelper && matchesRegistry(p, registry)
	})
	if err != nil {
		return nil, err
	}
	var previous []string
	alreadyRecorded := false
	for _, p := range recorded {
		if p.Name == name {
			alreadyRecorded = true
		} else {
			previous = append(previous, p.Name)
		}
	}
	if alreadyRecorded && len(previous) == 0 {
		return nil, nil
	}
	if err := databrickscfg.SetDockerCredentialHelperProfile(ctx, name, env.Get(ctx, "DATABRICKS_CONFIG_FILE"), previous); err != nil {
		return nil, err
	}
	return previous, nil
}

// ProfileForRegistry returns the Docker credential helper profile for registry.
// A profile recorded by RecordProfile takes precedence, even when it is no longer supported.
func ProfileForRegistry(ctx context.Context, profiler profile.Profiler, registry Registry) (profile.Profile, error) {
	workspaceProfiles, err := profiler.LoadProfiles(ctx, func(p profile.Profile) bool {
		return matchesRegistry(p, registry)
	})
	if err != nil {
		return profile.Profile{}, err
	}
	if len(workspaceProfiles) == 0 {
		return profile.Profile{}, fmt.Errorf("no Databricks profile found for workspace ID %s from registry host %s. Run databricks auth login --host <workspace-url> and set workspace_id for that profile", registry.WorkspaceID, registry.Host)
	}

	var recorded, matchingProfiles profile.Profiles
	for _, p := range workspaceProfiles {
		if p.DockerCredentialHelper {
			recorded = append(recorded, p)
		}
		if ValidateProfile(p) == nil {
			matchingProfiles = append(matchingProfiles, p)
		}
	}
	switch {
	case len(recorded) == 1:
		if err := ValidateProfile(recorded[0]); err != nil {
			return profile.Profile{}, err
		}
		return recorded[0], nil
	case len(recorded) > 1:
		return profile.Profile{}, duplicateProfilesError(registry.WorkspaceID, recorded.Names())
	case len(matchingProfiles) == 0:
		return profile.Profile{}, ValidateProfile(workspaceProfiles[0])
	case len(matchingProfiles) == 1:
		return matchingProfiles[0], nil
	default:
		return profile.Profile{}, duplicateProfilesError(registry.WorkspaceID, matchingProfiles.Names())
	}
}

func matchesRegistry(p profile.Profile, registry Registry) bool {
	return p.WorkspaceID == registry.WorkspaceID && registry.ServesWorkspaceHost(p.Host)
}

func duplicateProfilesError(workspaceID string, names []string) error {
	return fmt.Errorf("multiple Databricks profiles match workspace ID %s: %s. Select the profile for Docker with: databricks auth docker configure <profile-name>", workspaceID, strings.Join(names, ", "))
}

// RewriteProfileError adds Docker profile context to authentication errors.
func RewriteProfileError(ctx context.Context, p profile.Profile, err error) error {
	if rewritten, rewrittenErr := auth.RewriteAuthError(ctx, p.Host, p.AccountID, p.Name, err); rewritten {
		return rewrittenErr
	}
	return err
}
