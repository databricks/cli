package databrickscfg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"gopkg.in/ini.v1"
)

const fileMode = 0o600

const defaultComment = "The profile defined in the DEFAULT section is to be used as a fallback when no profile is explicitly specified."

func loadOrCreateConfigFile(filename string) (*config.File, error) {
	if filename == "" {
		filename = "~/.databrickscfg"
	}
	// Expand ~ to home directory, as we need a deterministic name for os.OpenFile
	// to work in the cases when ~/.databrickscfg does not exist yet
	if strings.HasPrefix(filename, "~") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot find homedir: %w", err)
		}
		filename = fmt.Sprintf("%s%s", homedir, filename[1:])
	}
	configFile, err := config.LoadFile(filename)
	if err != nil && os.IsNotExist(err) {
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
		if cfg.Profile == s.Name() {
			return true
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

func SaveToProfile(ctx context.Context, cfg *config.Config) error {
	configFile, err := loadOrCreateConfigFile(cfg.ConfigFile)
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

	// clear old keys in case we're overriding the section
	for _, oldKey := range section.KeyStrings() {
		section.DeleteKey(oldKey)
	}

	for _, attr := range config.ConfigAttributes {
		if attr.IsZero(cfg) {
			continue
		}
		key := section.Key(attr.Name)
		key.SetValue(attr.GetString(cfg))
	}

	// Add a comment to the default section if it's empty.
	section = configFile.Section(ini.DefaultSection)
	if len(section.Keys()) == 0 && section.Comment == "" {
		section.Comment = defaultComment
	}

	orig, backupErr := os.ReadFile(configFile.Path())
	if len(orig) > 0 && backupErr == nil {
		log.Infof(ctx, "Backing up in %s.bak", configFile.Path())
		err = os.WriteFile(configFile.Path()+".bak", orig, fileMode)
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

func init() {
	// We document databrickscfg files with a [DEFAULT] header and wish to keep it that way.
	// This, however, does mean we emit a [DEFAULT] section even if it's empty.
	ini.DefaultHeader = true
}
