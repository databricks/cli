package dockercredentials

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
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
	return p.AccountID != "" && (p.WorkspaceID == "" || p.WorkspaceID == auth.WorkspaceIDNone)
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

// EnsureUniqueProfile verifies that workspaceID resolves to only p.
func EnsureUniqueProfile(ctx context.Context, profiler profile.Profiler, p profile.Profile, workspaceID string) error {
	matches, err := profiler.LoadProfiles(ctx, func(candidate profile.Profile) bool {
		return candidate.WorkspaceID == workspaceID
	})
	if err != nil {
		return err
	}

	if p.WorkspaceID == "" || p.WorkspaceID == auth.WorkspaceIDNone {
		matches = append(matches, p)
	}
	var names []string
	for _, candidate := range matches {
		if ValidateProfile(candidate) == nil {
			names = append(names, candidate.Name)
		}
	}
	if len(names) <= 1 {
		return nil
	}

	return fmt.Errorf("multiple Databricks profiles match workspace ID %s: %s. Remove duplicate workspace_id entries before using Docker credential helper", workspaceID, strings.Join(names, " and "))
}

// ProfileForRegistry returns the Docker credential helper profile for registry.
func ProfileForRegistry(ctx context.Context, profiler profile.Profiler, registry Registry) (profile.Profile, error) {
	workspaceProfiles, err := profiler.LoadProfiles(ctx, func(p profile.Profile) bool {
		return p.WorkspaceID == registry.WorkspaceID && registry.ServesWorkspaceHost(p.Host)
	})
	if err != nil {
		return profile.Profile{}, err
	}
	if len(workspaceProfiles) == 0 {
		return profile.Profile{}, fmt.Errorf("no Databricks profile found for workspace ID %s from registry host %s. Run databricks auth login --host <workspace-url> and set workspace_id for that profile", registry.WorkspaceID, registry.Host)
	}

	var matchingProfiles profile.Profiles
	for _, p := range workspaceProfiles {
		if ValidateProfile(p) == nil {
			matchingProfiles = append(matchingProfiles, p)
		}
	}
	switch len(matchingProfiles) {
	case 0:
		return profile.Profile{}, ValidateProfile(workspaceProfiles[0])
	case 1:
		return matchingProfiles[0], nil
	default:
		return profile.Profile{}, fmt.Errorf("multiple Databricks profiles match workspace ID %s: %s. Remove duplicate workspace_id entries before using Docker credential helper", registry.WorkspaceID, strings.Join(matchingProfiles.Names(), " and "))
	}
}

// RewriteProfileError adds Docker profile context to authentication errors.
func RewriteProfileError(ctx context.Context, p profile.Profile, err error) error {
	if rewritten, rewrittenErr := auth.RewriteAuthError(ctx, p.Host, p.AccountID, p.Name, err); rewritten {
		return rewrittenErr
	}
	return err
}
