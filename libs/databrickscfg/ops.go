package databrickscfg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filelock"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"gopkg.in/ini.v1"
)

const fileMode = 0o600

const defaultComment = "The profile defined in the DEFAULT section is to be used as a fallback when no profile is explicitly specified."

const (
	databricksSettingsSection = "__settings__"
	defaultProfileKey         = "default_profile"
	authStorageKey            = "auth_storage"
)

// GetConfiguredDefaultProfile returns the explicitly configured default profile
// by loading the config file at configFilePath.
// Returns "" if the file doesn't exist or default_profile is not set.
func GetConfiguredDefaultProfile(ctx context.Context, configFilePath string) (string, error) {
	configFile, err := loadConfigFile(ctx, configFilePath)
	if err != nil {
		return "", err
	}
	if configFile == nil {
		return "", nil
	}
	return GetConfiguredDefaultProfileFrom(configFile), nil
}

// ResolveDefaultProfile returns the default profile from the config file
// pointed to by DATABRICKS_CONFIG_FILE (or ~/.databrickscfg when unset):
// [__settings__].default_profile, else [DEFAULT] if it has a host key,
// else "". Returns "" with no error when the file is missing or parsing
// fails (a warning is logged on parse error).
//
// Callers must respect their own higher-priority sources (an explicit
// --profile flag or DATABRICKS_CONFIG_PROFILE env var) before consulting
// this helper. default_profile and [DEFAULT] are CLI-level fallbacks; the
// SDK loader silently falls back to [DEFAULT] but leaves cfg.Profile empty,
// which breaks the per-profile OAuth cache key. Pinning the name here keeps
// cfg.Profile in sync with what the SDK would read.
//
// Single-profile fallback (using "the only profile in the file" as the
// default) is intentionally NOT applied: that is a prompt-seeding convenience
// (see GetDefaultProfile), not an auth rule, and it would silently route a
// single account-only profile through the workspace-client path.
func ResolveDefaultProfile(ctx context.Context) string {
	configFilePath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	configFile, err := loadConfigFile(ctx, configFilePath)
	if err != nil {
		log.Warnf(ctx, "Failed to load default profile: %v", err)
		return ""
	}
	if configFile == nil {
		return ""
	}
	if profile := GetConfiguredDefaultProfileFrom(configFile); profile != "" {
		log.Debugf(ctx, "profile %q resolved from [__settings__].default_profile", profile)
		return profile
	}
	if section := configFile.Section(ini.DefaultSection); section.HasKey("host") {
		log.Debugf(ctx, "profile %q resolved from the [DEFAULT] section", ini.DefaultSection)
		return ini.DefaultSection
	}
	return ""
}

// GetConfiguredDefaultProfileFrom returns the explicit default profile from
// [__settings__].default_profile, or "" when it is not set or when the value
// is the reserved __settings__ section name itself.
func GetConfiguredDefaultProfileFrom(configFile *config.File) string {
	v := configFile.Section(databricksSettingsSection).Key(defaultProfileKey).String()
	if v == databricksSettingsSection {
		return ""
	}
	return v
}

// GetConfiguredAuthStorage returns the explicitly configured auth_storage
// value from [__settings__].auth_storage, or "" if not set. Loads the config
// file at configFilePath. Returns "" (not an error) when the file does not
// exist.
func GetConfiguredAuthStorage(ctx context.Context, configFilePath string) (string, error) {
	configFile, err := loadConfigFile(ctx, configFilePath)
	if err != nil {
		return "", err
	}
	if configFile == nil {
		return "", nil
	}
	return GetConfiguredAuthStorageFrom(configFile), nil
}

// GetConfiguredAuthStorageFrom returns [__settings__].auth_storage from an
// already-loaded config file, or "" when not set.
func GetConfiguredAuthStorageFrom(configFile *config.File) string {
	return configFile.Section(databricksSettingsSection).Key(authStorageKey).String()
}

// GetDefaultProfile returns the name of the default profile by loading the
// config file at configFilePath. Returns "" if the file doesn't exist.
// See GetDefaultProfileFrom for resolution order.
func GetDefaultProfile(ctx context.Context, configFilePath string) (string, error) {
	configFile, err := loadConfigFile(ctx, configFilePath)
	if err != nil {
		return "", err
	}
	if configFile == nil {
		return "", nil
	}
	return GetDefaultProfileFrom(configFile), nil
}

// loadConfigFile loads a config file without creating it if it doesn't exist.
// Returns (nil, nil) when the file is not found.
func loadConfigFile(ctx context.Context, filename string) (*config.File, error) {
	filename, err := resolveConfigFilePath(ctx, filename)
	if err != nil {
		return nil, err
	}
	configFile, err := config.LoadFile(filename)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	return configFile, nil
}

// resolveConfigFilePath defaults to ~/.databrickscfg and expands ~ to the home directory.
func resolveConfigFilePath(ctx context.Context, filename string) (string, error) {
	if filename == "" {
		filename = "~/.databrickscfg"
	}
	if strings.HasPrefix(filename, "~") {
		homedir, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", fmt.Errorf("cannot find homedir: %w", err)
		}

		filename = fmt.Sprintf("%s%s", homedir, filename[1:])
	}
	return filename, nil
}

func resolveConfigTarget(ctx context.Context, filename string) (string, error) {
	resolved, err := resolveConfigFilePath(ctx, filename)
	if err != nil {
		return "", err
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", resolved, err)
	}
	target, err := filepath.EvalSymlinks(resolved)
	if err == nil {
		return target, nil
	}
	if _, lstatErr := os.Lstat(resolved); !errors.Is(lstatErr, fs.ErrNotExist) {
		return "", fmt.Errorf("resolve config file %s: %w", resolved, err)
	}
	dir, dirErr := filepath.EvalSymlinks(filepath.Dir(resolved))
	if dirErr != nil {
		return "", fmt.Errorf("resolve config directory for %s: %w", resolved, dirErr)
	}
	return filepath.Join(dir, filepath.Base(resolved)), nil
}

func withConfigFileLock(ctx context.Context, filename string, fn func(string) error) (err error) {
	target, err := resolveConfigTarget(ctx, filename)
	if err != nil {
		return err
	}
	release, err := filelock.Acquire(ctx, target+".lock", time.Minute)
	if err != nil {
		return fmt.Errorf("lock config file %s: %w", target, err)
	}
	defer func() {
		err = errors.Join(err, release())
	}()
	return fn(target)
}

func mutateConfigFile(ctx context.Context, filename string, fn func(*config.File) error) error {
	return withConfigFileLock(ctx, filename, func(target string) error {
		configFile, err := loadOrCreateConfigFile(ctx, target)
		if err != nil {
			return err
		}
		if err := fn(configFile); err != nil {
			return err
		}
		return writeConfigFile(ctx, configFile)
	})
}

// GetDefaultProfileFrom returns the name of the default profile from an
// already-loaded config file. It uses the following resolution order:
//  1. Explicit default_profile key in [__settings__].
//  2. If there is exactly one profile in the file, return it.
//  3. If a profile named DEFAULT exists, return it.
//  4. Empty string (no default).
func GetDefaultProfileFrom(configFile *config.File) string {
	// 1. Check for explicit default_profile setting.
	if profile := GetConfiguredDefaultProfileFrom(configFile); profile != "" {
		return profile
	}

	// Collect profile sections (sections that have a "host" key, excluding
	// the settings section).
	var profileNames []string
	hasDefault := false
	for _, s := range configFile.Sections() {
		if s.Name() == databricksSettingsSection {
			continue
		}
		if !s.HasKey("host") {
			continue
		}
		profileNames = append(profileNames, s.Name())
		if s.Name() == ini.DefaultSection {
			hasDefault = true
		}
	}

	// 2. Exactly one profile: treat it as the default.
	if len(profileNames) == 1 {
		return profileNames[0]
	}

	// 3. Legacy fallback: a DEFAULT section with a host key.
	if hasDefault {
		return ini.DefaultSection
	}

	return ""
}

// isFirstProfileInFile returns true if the config file has no profiles (sections with a "host" key) yet.
func isFirstProfileInFile(configFile *config.File) bool {
	for _, s := range configFile.Sections() {
		if s.Name() == databricksSettingsSection {
			continue
		}
		if s.HasKey("host") {
			return false
		}
	}
	return true
}

// SetDefaultProfile writes the default_profile key to the [__settings__] section.
func SetDefaultProfile(ctx context.Context, profileName, configFilePath string) error {
	if profileName == databricksSettingsSection {
		return fmt.Errorf("profile name %q is reserved for internal use", databricksSettingsSection)
	}
	return mutateConfigFile(ctx, configFilePath, func(configFile *config.File) error {
		section, err := configFile.GetSection(databricksSettingsSection)
		if err != nil {
			section, err = configFile.NewSection(databricksSettingsSection)
			if err != nil {
				return fmt.Errorf("cannot create %s section: %w", databricksSettingsSection, err)
			}
		}
		section.Key(defaultProfileKey).SetValue(profileName)
		return nil
	})
}

// SetConfiguredAuthStorage writes the auth_storage key to the [__settings__]
// section. Used by auth login to persist a plaintext fallback when the OS
// keyring is unreachable, so subsequent commands skip the keyring probe and
// route directly to the file cache.
func SetConfiguredAuthStorage(ctx context.Context, value, configFilePath string) error {
	return mutateConfigFile(ctx, configFilePath, func(configFile *config.File) error {
		section, err := configFile.GetSection(databricksSettingsSection)
		if err != nil {
			section, err = configFile.NewSection(databricksSettingsSection)
			if err != nil {
				return fmt.Errorf("cannot create %s section: %w", databricksSettingsSection, err)
			}
		}
		section.Key(authStorageKey).SetValue(value)
		return nil
	})
}

// SaveResourcesToProfile writes (or clears) the `resources` key on a profile
// section. New values are JSON string arrays so commas and whitespace inside
// individual RFC 8707 resource indicators remain significant. An empty list
// removes the key. The profile section must already exist.
func SaveResourcesToProfile(ctx context.Context, profileName, configFilePath string, resources []string) error {
	return mutateConfigFile(ctx, configFilePath, func(configFile *config.File) error {
		section, err := configFile.GetSection(profileName)
		if err != nil {
			return fmt.Errorf("profile %q not found: %w", profileName, err)
		}
		if len(resources) == 0 {
			section.DeleteKey("resources")
			return nil
		}
		encoded, err := json.Marshal(resources)
		if err != nil {
			return fmt.Errorf("encode RFC 8707 resources: %w", err)
		}
		section.Key("resources").SetValue(string(encoded))
		return nil
	})
}

// ClearDefaultProfile removes the default_profile key from the [__settings__]
// section if the current default matches the given profile name.
func ClearDefaultProfile(ctx context.Context, profileName, configFilePath string) error {
	return withConfigFileLock(ctx, configFilePath, func(target string) error {
		configFile, err := loadConfigFile(ctx, target)
		if err != nil || configFile == nil {
			return err
		}
		if GetConfiguredDefaultProfileFrom(configFile) != profileName {
			return nil
		}
		section, err := configFile.GetSection(databricksSettingsSection)
		if err != nil {
			return nil //nolint:nilerr
		}
		section.DeleteKey(defaultProfileKey)
		return writeConfigFile(ctx, configFile)
	})
}

func loadOrCreateConfigFile(ctx context.Context, filename string) (*config.File, error) {
	filename, err := resolveConfigFilePath(ctx, filename)
	if err != nil {
		return nil, err
	}
	configFile, err := config.LoadFile(filename)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", filename, err)
		}
		defer file.Close()
		configFile, err = config.LoadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("load created %s: %w", filename, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	return configFile, nil
}

func matchOrCreateSection(ctx context.Context, configFile *config.File, cfg *config.Config) (*ini.Section, error) {
	section, err := findMatchingProfile(configFile, func(s *ini.Section) bool {
		if cfg.Profile != "" {
			return cfg.Profile == s.Name()
		}
		raw := s.KeysHash()
		if cfg.AccountID != "" {
			// here we rely on map zerovals for matching with accounts:
			// if profile has no account id, the raw["account_id"] will be empty
			return cfg.AccountID == raw["account_id"]
		}
		if cfg.Host == "" {
			return false
		}
		host, ok := raw["host"]
		if !ok {
			log.Tracef(ctx, "section %s: no host", s.Name())
			return false
		}
		// Check if this section matches the normalized host
		return normalizeHost(host) == normalizeHost(cfg.Host)
	})
	if err == errNoMatchingProfiles {
		section, err = configFile.NewSection(cfg.Profile)
		if err != nil {
			return nil, fmt.Errorf("cannot create new profile: %w", err)
		}
	} else if err != nil {
		return nil, err
	}
	return section, nil
}

// ExperimentalIsUnifiedHostKey is the INI key for the deprecated
// experimental_is_unified_host flag. Unified hosts are now detected from
// /.well-known/databricks-config; the key is only ever cleared from profiles
// (never read or written) so stale values don't influence routing.
const ExperimentalIsUnifiedHostKey = "experimental_is_unified_host"

// AuthCredentialKeys returns the config file key names for all auth credential
// fields from the SDK's ConfigAttributes. These are fields annotated with an
// auth type (e.g. pat, basic, oauth, azure, google). Use this to clear stale
// credentials when switching auth methods.
func AuthCredentialKeys() []string {
	var keys []string
	for _, attr := range config.ConfigAttributes {
		if attr.HasAuthAttribute() {
			keys = append(keys, attr.Name)
		}
	}
	return keys
}

// We document databrickscfg files with a [DEFAULT] header and wish to keep it that way.
// This, however, does mean we emit a [DEFAULT] section even if it's empty.
// ini.DefaultHeader is process-global, so it is enabled on first write instead of
// at import time to keep this package free of import side effects.
var enableDefaultHeader = sync.OnceFunc(func() {
	ini.DefaultHeader = true
})

var writeAtomicFile = atomicfile.Write

func writeConfigFile(ctx context.Context, configFile *config.File) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	enableDefaultHeader()
	section := configFile.Section(ini.DefaultSection)
	if len(section.Keys()) == 0 && section.Comment == "" {
		section.Comment = defaultComment
	}

	var rendered bytes.Buffer
	if _, err := configFile.WriteTo(&rendered); err != nil {
		return fmt.Errorf("render %s: %w", configFile.Path(), err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	orig, err := os.ReadFile(configFile.Path())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %s for backup: %w", configFile.Path(), err)
	}
	if len(orig) > 0 {
		log.Infof(ctx, "Backing up in %s.bak", configFile.Path())
		if err := writeAtomicFile(configFile.Path()+".bak", orig, fileMode, atomicfile.PreserveMode()); err != nil {
			return fmt.Errorf("backup: %w", err)
		}
		log.Infof(ctx, "Overwriting %s", configFile.Path())
	} else {
		log.Infof(ctx, "Saving %s", configFile.Path())
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writeAtomicFile(configFile.Path(), rendered.Bytes(), fileMode, atomicfile.PreserveMode()); err != nil {
		return fmt.Errorf("write %s: %w", configFile.Path(), err)
	}
	return nil
}

// SaveToProfile merges the provided config into a .databrickscfg profile.
// Non-zero fields in cfg overwrite existing values. Existing keys not
// mentioned in cfg are preserved. Keys listed in clearKeys are explicitly
// removed (use this for mutually exclusive fields like cluster_id vs
// serverless_compute_id, or to drop stale auth credentials on auth-type switch).
func SaveToProfile(ctx context.Context, cfg *config.Config, clearKeys ...string) error {
	if cfg.Profile == databricksSettingsSection {
		return fmt.Errorf("profile name %q is reserved for internal use", databricksSettingsSection)
	}
	profileName := cfg.Profile
	configFilePath := cfg.ConfigFile
	return mutateConfigFile(ctx, configFilePath, func(configFile *config.File) error {
		firstProfile := isFirstProfileInFile(configFile)
		section, err := matchOrCreateSection(ctx, configFile, cfg)
		if err != nil {
			return err
		}
		cfg.Profile = ""
		cfg.ConfigFile = ""
		defer func() {
			cfg.Profile = profileName
			cfg.ConfigFile = configFilePath
		}()
		for _, key := range clearKeys {
			section.DeleteKey(key)
		}
		for _, attr := range config.ConfigAttributes {
			if attr.IsZero(cfg) {
				continue
			}
			section.Key(attr.Name).SetValue(attr.GetString(cfg))
		}
		if firstProfile && profileName != "" {
			configFile.Section(databricksSettingsSection).Key(defaultProfileKey).SetValue(profileName)
			log.Debugf(ctx, "Auto-setting default profile to %q (first profile)", profileName)
		}
		return nil
	})
}

// DeleteProfile removes the named profile section from the databrickscfg file.
// It creates a backup of the original file before modifying it.
func DeleteProfile(ctx context.Context, profileName, configFilePath string) error {
	return withConfigFileLock(ctx, configFilePath, func(target string) error {
		configFile, err := config.LoadFile(target)
		if err != nil {
			return fmt.Errorf("cannot load config file %s: %w", target, err)
		}
		if _, err := configFile.SectionsByName(profileName); err != nil {
			return fmt.Errorf("profile %q not found: %w", profileName, err)
		}
		if profileName == ini.DefaultSection {
			section := configFile.Section(ini.DefaultSection)
			for _, key := range section.Keys() {
				section.DeleteKey(key.Name())
			}
		} else {
			configFile.DeleteSection(profileName)
		}
		return writeConfigFile(ctx, configFile)
	})
}

func ValidateConfigAndProfileHost(cfg *config.Config, profile string) error {
	configFile, err := config.LoadFile(cfg.ConfigFile)
	if err != nil {
		return fmt.Errorf("cannot parse config file: %w", err)
	}

	// Normalized version of the configured host.
	host := normalizeHost(cfg.Host)
	match, err := findMatchingProfile(configFile, func(s *ini.Section) bool {
		return profile == s.Name()
	})
	if err != nil {
		return err
	}

	hostFromProfile := normalizeHost(match.Key("host").Value())
	if hostFromProfile != "" && host != "" && hostFromProfile != host {
		// Try to find if there's a profile which uses the same host as the bundle and suggest in error message
		match, err = findMatchingProfile(configFile, func(s *ini.Section) bool {
			return normalizeHost(s.Key("host").Value()) == host
		})
		if err == nil && match != nil {
			profileName := match.Name()
			return fmt.Errorf("the host in the profile (%s) doesn’t match the host configured in the bundle (%s). The profile \"%s\" has host=\"%s\" that matches host in the bundle. To select it, pass \"-p %s\"", hostFromProfile, host, profileName, host, profileName)
		}

		return fmt.Errorf("the host in the profile (%s) doesn’t match the host configured in the bundle (%s)", hostFromProfile, host)
	}

	return nil
}
