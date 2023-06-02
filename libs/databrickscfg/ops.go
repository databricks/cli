package databrickscfg

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"gopkg.in/ini.v1"
)

const fileMode = 0o600

func loadOrCreateConfigFile(filename string) (*config.File, error) {
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

	// ignoring err because we've read the file already
	orig, backupErr := os.ReadFile(configFile.Path())
	if len(orig) > 0 && backupErr == nil {
		log.Infof(ctx, "Backing up in %s.bak", configFile.Path())
		err = os.WriteFile(configFile.Path()+".bak", orig, fileMode)
		if err != nil {
			return fmt.Errorf("backup: %w", err)
		}
		log.Infof(ctx, "Overriding %s", configFile.Path())
	} else {
		log.Infof(ctx, "Saving %s", configFile.Path())
	}
	return configFile.SaveTo(configFile.Path())
}
