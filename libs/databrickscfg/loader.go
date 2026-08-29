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

// ResolveNonAuthFromEnv reads non-auth, non-host config from environment
// variables. See ProfileAuthLoaders for how it fits into the chain.
var ResolveNonAuthFromEnv = envLoader{name: "environment (excluding auth)", skipAuth: true}

// resolveAuthGapFromEnv gap-fills the auth attributes and host the profile left
// empty from environment variables. See ProfileAuthLoaders for how it fits into
// the chain.
var resolveAuthGapFromEnv = envLoader{name: "environment (auth gap-fill)", skipAuth: false}

// ProfileAuthLoaders is the loader chain for an explicitly selected profile
// (--profile or a bundle's workspace.profile). Unlike the SDK's default
// env-first chain, the profile wins over auth env vars (#5096):
//
//  1. ResolveNonAuthFromEnv: non-auth env attrs (e.g. cluster_id), env-wins.
//  2. ConfigFile: the selected profile (host, routing, auth).
//  3. resolveAuthGapFromEnv: gap-fills the auth attrs and host the profile left
//     empty from env, so a host-less profile still reaches the env host and a
//     complete conflicting env auth method surfaces as an error rather than being
//     silently dropped. It skips envAlwaysSkipAttrs so routing/auth-steering
//     attrs come from the profile only; using the SDK's ConfigAttributes here
//     would let those env vars shadow the profile (e.g. DATABRICKS_AUTH_TYPE=basic
//     over a PAT profile).
//
// A profile from DATABRICKS_CONFIG_PROFILE keeps env-first precedence; only an
// out-of-band profile name triggers this chain.
var ProfileAuthLoaders = []config.Loader{
	ResolveNonAuthFromEnv,
	config.ConfigFile,
	resolveAuthGapFromEnv,
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

// hostAttr is the SDK config attribute name for the workspace/account host.
const hostAttr = "host"

// envAlwaysSkipAttrs lists env attributes envLoader must never read when a
// profile is explicitly selected, in either pass: routing (workspace_id,
// account_id) and auth-steering fields tagged auth:"-" (auth_type, discovery_url,
// audience, cloud), which HasAuthAttribute misses. Reading these from env — even
// as a gap-fill — would re-steer or shadow the selected profile's auth (#5096).
// The host is deliberately absent: like the auth attrs it is skipped only in the
// env-first pass (see envLoader), then gap-filled after the profile so a
// host-less profile falls back to the env host.
// TestEnvAlwaysSkipAttrsCoverSDKInternalEnvAttrs guards against SDK drift.
var envAlwaysSkipAttrs = map[string]bool{
	"workspace_id":  true,
	"account_id":    true,
	"auth_type":     true,
	"discovery_url": true,
	"audience":      true,
	"cloud":         true,
}

// envLoader reads config attributes from environment variables. It always skips
// envAlwaysSkipAttrs so routing/auth-steering attrs come from the selected
// profile only (#5096). When skipAuth is true it additionally skips the SDK's
// auth attributes and the host, for the env-first pass that runs before the
// profile; the gap-fill pass then fills the ones the profile left empty.
type envLoader struct {
	name     string
	skipAuth bool
}

func (l envLoader) Name() string {
	return l.name
}

func (l envLoader) Configure(cfg *config.Config) error {
	for _, attr := range config.ConfigAttributes {
		// Routing/auth-steering come from the profile (config file), not env.
		if envAlwaysSkipAttrs[attr.Name] {
			continue
		}
		// The env-first pass leaves the auth attrs and host for the profile; the
		// gap-fill pass runs after the profile and fills only those it left empty.
		if l.skipAuth && (attr.HasAuthAttribute() || attr.Name == hostAttr) {
			continue
		}
		// Don't overwrite an already-set value (SDK loader semantics).
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
		// Record the source so `auth describe` attributes the value to env.
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
