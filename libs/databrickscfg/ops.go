package databrickscfg

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"gopkg.in/ini.v1"
)

const fileMode = 0o600

const defaultComment = "The profile defined in the DEFAULT section is to be used as a fallback when no profile is explicitly specified."

const databricksSettingsSection = "__settings__"

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

// GetConfiguredDefaultProfileFrom returns the explicit default profile from
// [__settings__].default_profile, or "" when it is not set.
func GetConfiguredDefaultProfileFrom(configFile *config.File) string {
	section, err := configFile.GetSection(databricksSettingsSection)
	if err != nil {
		return ""
	}
	key, err := section.GetKey("default_profile")
	if err != nil {
		return ""
	}
	return key.String()
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

// SetDefaultProfile writes the default_profile key to the [__settings__] section.
func SetDefaultProfile(ctx context.Context, profileName, configFilePath string) error {
	configFile, err := loadOrCreateConfigFile(ctx, configFilePath)
	if err != nil {
		return err
	}

	section, err := configFile.GetSection(databricksSettingsSection)
	if err != nil {
		// Section doesn't exist, create it.
		section, err = configFile.NewSection(databricksSettingsSection)
		if err != nil {
			return fmt.Errorf("cannot create %s section: %w", databricksSettingsSection, err)
		}
	}

	section.Key("default_profile").SetValue(profileName)

	return backupAndSaveConfigFile(ctx, configFile)
}

// backupAndSaveConfigFile adds a default section comment if needed, creates
// a .bak backup of the existing file, and saves the config file to disk.
func backupAndSaveConfigFile(ctx context.Context, configFile *config.File) error {
	// Add a comment to the default section if it's empty.
	section := configFile.Section(ini.DefaultSection)
	if len(section.Keys()) == 0 && section.Comment == "" {
		section.Comment = defaultComment
	}

	orig, backupErr := os.ReadFile(configFile.Path())
	if len(orig) > 0 && backupErr == nil {
		log.Infof(ctx, "Backing up in %s.bak", configFile.Path())
		err := os.WriteFile(configFile.Path()+".bak", orig, fileMode)
		if err != nil {
			return fmt.Errorf("backup: %w", err)
		}
		log.Infof(ctx, "Overwriting %s", configFile.Path())
	} else if backupErr != nil {
		log.Warnf(ctx, "Failed to backup %s: %v. Proceeding to save",
			configFile.Path(), backupErr)
	} else {
		log.Infof(ctx, "Saving %s", configFile.Path())
	}
	return configFile.SaveTo(configFile.Path())
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

// SaveToProfile merges the provided config into a .databrickscfg profile.
// Non-zero fields in cfg overwrite existing values. Existing keys not
// mentioned in cfg are preserved. Keys listed in clearKeys are explicitly
// removed (use this for mutually exclusive fields like cluster_id vs
// serverless_compute_id, or to drop stale auth credentials on auth-type switch).
func SaveToProfile(ctx context.Context, cfg *config.Config, clearKeys ...string) error {
	configFile, err := loadOrCreateConfigFile(ctx, cfg.ConfigFile)
	if err != nil {
		return err
	}

	section, err := matchOrCreateSection(ctx, configFile, cfg)
	if err != nil {
		return err
	}

	// zeroval profile name before adding it to a section
	cfg.Profile = ""
	cfg.ConfigFile = ""

	// Explicitly remove keys the caller wants cleared.
	for _, key := range clearKeys {
		section.DeleteKey(key)
	}

	// Write non-zero fields from the new config. Iterates ConfigAttributes
	// in declaration order for deterministic key ordering on new profiles.
	for _, attr := range config.ConfigAttributes {
		if attr.IsZero(cfg) {
			continue
		}
		key := section.Key(attr.Name)
		key.SetValue(attr.GetString(cfg))
	}

	return backupAndSaveConfigFile(ctx, configFile)
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

func init() {
	// We document databrickscfg files with a [DEFAULT] header and wish to keep it that way.
	// This, however, does mean we emit a [DEFAULT] section even if it's empty.
	ini.DefaultHeader = true
}
