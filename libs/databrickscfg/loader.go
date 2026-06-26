package databrickscfg

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"gopkg.in/ini.v1"
)

var ResolveProfileFromHost = profileFromHostLoader{}

// ResolveNonAuthFromEnv reads config from environment variables, except for the
// host and any auth credential. It runs before the config file loader when a
// profile is explicitly selected, so the profile wins over auth env vars: the
// SDK's default chain reads env before the config file and never overwrites a
// set field, which would otherwise let env vars shadow the profile (#5096).
var ResolveNonAuthFromEnv = nonAuthEnvLoader{}

// ProfileAuthLoaders is the loader chain to use when a profile is explicitly
// selected (--profile or a bundle's workspace.profile), and the single source
// of truth for that precedence rule. The profile must win for host, routing and
// auth over the matching env vars (DATABRICKS_HOST, DATABRICKS_TOKEN, ...),
// which the SDK's default env-first chain would otherwise let shadow it (#5096).
//
// This only governs an explicitly selected profile. One picked up from
// DATABRICKS_CONFIG_PROFILE keeps env-first precedence: reordering two
// environment signals (DATABRICKS_CONFIG_PROFILE vs DATABRICKS_HOST) is the
// SDK's domain; we only override when the profile is named out-of-band.
//
// Order:
//  1. ResolveNonAuthFromEnv: non-auth, non-routing env attrs (e.g. cluster_id),
//     keeping env-wins precedence for those.
//  2. ConfigFile: the selected profile (host, routing, auth).
//  3. ConfigAttributes: gap-fills only fields the profile left empty (e.g. a
//     host-only profile + DATABRICKS_TOKEN, a common CI pattern); it never
//     overwrites a profile value, so the profile still wins.
var ProfileAuthLoaders = []config.Loader{
	ResolveNonAuthFromEnv,
	config.ConfigFile,
	config.ConfigAttributes,
}

var errNoMatchingProfiles = errors.New("no matching config profiles found")

type errMultipleProfiles []string

func (e errMultipleProfiles) Error() string {
	return "multiple profiles matched: " + strings.Join(e, ", ")
}

// AsMultipleProfiles checks if the error is caused by multiple profiles
// matching the same host. If so, it returns the matching profile names.
func AsMultipleProfiles(err error) ([]string, bool) {
	if e, ok := errors.AsType[errMultipleProfiles](err); ok {
		return []string(e), true
	}
	return nil, false
}

func findMatchingProfile(configFile *config.File, matcher func(*ini.Section) bool) (*ini.Section, error) {
	// Look for sections in the configuration file that match the configured host.
	var matching []*ini.Section
	for _, section := range configFile.Sections() {
		if !matcher(section) {
			continue
		}
		matching = append(matching, section)
	}

	// If there are no matching sections, we don't do anything.
	if len(matching) == 0 {
		return nil, errNoMatchingProfiles
	}

	// If there are multiple matching sections, let the user know it is impossible
	// to unambiguously select a profile to use.
	if len(matching) > 1 {
		var names errMultipleProfiles
		for _, section := range matching {
			names = append(names, section.Name())
		}

		return nil, names
	}

	return matching[0], nil
}

// nonAuthEnvSkipAttrs lists SDK config attributes nonAuthEnvLoader must not read
// from the environment, beyond those caught by HasAuthAttribute. They identify
// the target workspace/account (host, routing IDs) or steer the auth method but
// are tagged auth:"-" (collapsed to Internal), so HasAuthAttribute misses them;
// leaving them to env would let an env var shadow the selected profile (#5096).
// Skipping only changes precedence: the trailing ConfigAttributes loader still
// gap-fills any the profile leaves empty.
//
//   - host: no `auth` tag at all.
//   - workspace_id / account_id: routing identifiers; an env var must not route
//     the profile's credentials elsewhere.
//   - auth_type: forces a specific auth method.
//   - discovery_url: redirects OIDC discovery.
//   - audience: selects the OIDC/workload-identity token audience.
//   - cloud: steers cloud-specific auth (Azure/GCP/AWS).
//
// Non-auth attrs tagged auth:"-" (oauth_callback_port, debug_headers, ...) are
// intentionally not skipped; TestNonAuthEnvSkipAttrsCoverSDKInternalEnvAttrs
// guards that every auth-steering internal attribute stays classified.
var nonAuthEnvSkipAttrs = map[string]bool{
	"host":          true,
	"workspace_id":  true,
	"account_id":    true,
	"auth_type":     true,
	"discovery_url": true,
	"audience":      true,
	"cloud":         true,
}

type nonAuthEnvLoader struct{}

func (nonAuthEnvLoader) Name() string {
	return "environment (excluding auth)"
}

func (nonAuthEnvLoader) Configure(cfg *config.Config) error {
	for _, attr := range config.ConfigAttributes {
		// Leave the host and authentication settings for the config file
		// (i.e. the selected profile) to provide.
		if nonAuthEnvSkipAttrs[attr.Name] || attr.HasAuthAttribute() {
			continue
		}
		// Match the SDK loader semantics: don't overwrite a value previously set.
		if !attr.IsZero(cfg) {
			continue
		}
		v, envName := attr.ReadEnv()
		if v == "" {
			continue
		}
		if err := attr.SetS(cfg, v); err != nil {
			return err
		}
		// Record the source so `databricks auth describe` and debug output
		// attribute the value to the environment, matching the SDK loader.
		cfg.SetAttrSource(&attr, config.Source{Type: config.SourceEnv, Name: envName})
	}
	return nil
}

type profileFromHostLoader struct{}

func (l profileFromHostLoader) Name() string {
	return "resolve-profile-from-host"
}

func (l profileFromHostLoader) Configure(cfg *config.Config) error {
	// Skip an attempt to resolve a profile from the host if any authentication
	// is already configured (either directly, through environment variables, or
	// if a profile was specified).
	if cfg.Host == "" || l.isAnyAuthConfigured(cfg) {
		return nil
	}

	ctx := context.Background() //nolint:gocritic // SDK interface does not accept context.
	configFile, err := config.LoadFile(cfg.ConfigFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cannot parse config file: %w", err)
	}
	// Normalized version of the configured host.
	host := normalizeHost(cfg.Host)
	match, err := findMatchingProfile(configFile, func(s *ini.Section) bool {
		key, err := s.GetKey("host")
		if err != nil {
			log.Tracef(ctx, "section %s: %s", s.Name(), err)
			return false
		}

		// Check if this section matches the normalized host
		return normalizeHost(key.Value()) == host
	})
	if err == errNoMatchingProfiles {
		return nil
	}

	// If multiple profiles match the same host and we have a workspace_id,
	// try to disambiguate by matching workspace_id.
	if names, ok := AsMultipleProfiles(err); ok && cfg.WorkspaceID != "" {
		originalErr := err
		match, err = l.disambiguateByWorkspaceID(ctx, configFile, host, cfg.WorkspaceID, names)
		if err == errNoMatchingProfiles {
			// workspace_id didn't match any of the host-matching profiles.
			// Fall back to the original ambiguity error.
			log.Debugf(ctx, "workspace_id=%s did not match any profiles for host %s: %v", cfg.WorkspaceID, host, names)
			err = originalErr
		}
	}

	if _, ok := AsMultipleProfiles(err); ok {
		return fmt.Errorf(
			"%s: %w: please set DATABRICKS_CONFIG_PROFILE or provide --profile flag to specify one",
			host, err)
	}
	if err != nil {
		return err
	}

	log.Debugf(ctx, "Loading profile %s because of host match", match.Name())
	err = config.ConfigAttributes.ResolveFromStringMapWithSource(cfg, match.KeysHash(), config.Source{
		Type: config.SourceFile,
		Name: configFile.Path(),
	})
	if err != nil {
		return fmt.Errorf("%s %s profile: %w", configFile.Path(), match.Name(), err)
	}

	cfg.Profile = match.Name()
	return nil
}

// disambiguateByWorkspaceID filters the profiles that matched a host by workspace_id.
func (l profileFromHostLoader) disambiguateByWorkspaceID(
	ctx context.Context,
	configFile *config.File,
	host string,
	workspaceID string,
	profileNames []string,
) (*ini.Section, error) {
	log.Debugf(ctx, "Multiple profiles matched host %s, disambiguating by workspace_id=%s", host, workspaceID)

	nameSet := make(map[string]bool, len(profileNames))
	for _, name := range profileNames {
		nameSet[name] = true
	}

	return findMatchingProfile(configFile, func(s *ini.Section) bool {
		if !nameSet[s.Name()] {
			return false
		}
		key, err := s.GetKey("workspace_id")
		if err != nil {
			return false
		}
		return key.Value() == workspaceID
	})
}

func (l profileFromHostLoader) isAnyAuthConfigured(cfg *config.Config) bool {
	// If any of the auth-specific attributes are set, we can skip profile resolution.
	for _, a := range config.ConfigAttributes {
		if !a.HasAuthAttribute() {
			continue
		}
		if !a.IsZero(cfg) {
			return true
		}
	}
	// If the auth type is set, we can skip profile resolution.
	// For example, to force "azure-cli", only the host and the auth type will be set.
	return cfg.AuthType != ""
}
