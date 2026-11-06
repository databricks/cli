package dockercredentials

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"golang.org/x/oauth2"
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

// TokenForRegistry returns a token from the first logged-in profile that ProfilesForRegistry returns.
// Only a missing or rejected login moves on to the next profile; any other error is returned.
func TokenForRegistry(ctx context.Context, profiler profile.Profiler, registry Registry, load func(context.Context, profile.Profile) (*oauth2.Token, error)) (*oauth2.Token, error) {
	profiles, err := ProfilesForRegistry(ctx, profiler, registry)
	if err != nil {
		return nil, err
	}
	// With one candidate there is nothing to fall back to, so its error is returned unchanged.
	if len(profiles) == 1 {
		return load(ctx, profiles[0])
	}
	var errs []error
	for _, p := range profiles {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token, err := load(ctx, p)
		if err == nil {
			return token, nil
		}
		err = fmt.Errorf("profile %q: %w", p.Name, err)
		if !isLoggedOut(err) {
			return nil, err
		}
		log.Debugf(ctx, "Skipping logged-out profile for registry %s: %v", registry.Host, err)
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("no logged-in Databricks profile for workspace ID %s from registry host %s. Run databricks auth login --profile <profile-name> for one of these profiles:\n%w", registry.WorkspaceID, registry.Host, errors.Join(errs...))
}

func isLoggedOut(err error) bool {
	_, invalidRefreshToken := errors.AsType[*u2m.InvalidRefreshTokenError](err)
	return invalidRefreshToken || errors.Is(err, storage.ErrNotFound) || errors.Is(err, u2m.ErrMissingRefreshToken)
}

// ProfilesForRegistry returns the profiles the Docker credential helper can use for registry, in profiler order.
func ProfilesForRegistry(ctx context.Context, profiler profile.Profiler, registry Registry) (profile.Profiles, error) {
	workspaceProfiles, err := profiler.LoadProfiles(ctx, func(p profile.Profile) bool {
		return p.WorkspaceID == registry.WorkspaceID && registry.ServesWorkspaceHost(p.Host)
	})
	if err != nil {
		return nil, err
	}
	if len(workspaceProfiles) == 0 {
		return nil, fmt.Errorf("no Databricks profile found for workspace ID %s from registry host %s. Run databricks auth login --host <workspace-url> and set workspace_id for that profile", registry.WorkspaceID, registry.Host)
	}

	var matchingProfiles profile.Profiles
	for _, p := range workspaceProfiles {
		if ValidateProfile(p) == nil {
			matchingProfiles = append(matchingProfiles, p)
		}
	}
	if len(matchingProfiles) == 0 {
		return nil, ValidateProfile(workspaceProfiles[0])
	}
	return matchingProfiles, nil
}

// RewriteProfileError adds Docker profile context to authentication errors.
func RewriteProfileError(ctx context.Context, p profile.Profile, err error) error {
	if rewritten, rewrittenErr := auth.RewriteAuthError(ctx, p.Host, p.AccountID, p.Name, err); rewritten {
		return rewrittenErr
	}
	return err
}
