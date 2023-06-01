package databrickscfg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"gopkg.in/ini.v1"
)

const fileMode = 0o600

func SaveToProfile(ctx context.Context, cfg *config.Config) error {
	configFile, err := config.LoadFile(cfg.ConfigFile)
	if err != nil && os.IsNotExist(err) {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot find homedir: %w", err)
		}
		path := filepath.Join(homedir, ".databrickscfg")
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
		if err != nil {
			return fmt.Errorf("create config file: %w", err)
		}
		err = file.Close()
		if err != nil {
			return fmt.Errorf("close config file: %w", err)
		}
		configFile, err = config.LoadFile(path)
		if err != nil {
			panic(fmt.Errorf("cannot load created file: %w", err))
		}
	} else if err != nil {
		return fmt.Errorf("cannot parse config file: %w", err)
	}

	section, err := findMatchingProfile(ctx, configFile, func(s *ini.Section) bool {
		if cfg.Profile == s.Name() {
			return true
		}
		raw := s.KeysHash()
		if cfg.AccountID != "" {
			// here we rely on map zerovals for matching with accounts
			return cfg.AccountID == raw["account_id"]
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
			return fmt.Errorf("cannot create new profile: %w", err)
		}
	} else if err != nil {
		return err
	}

	// zeroval profile name before adding it to a section
	cfg.Profile = ""

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
	orig, _ := os.ReadFile(configFile.Path())
	log.Infof(ctx, "Backing up in %s.bak", configFile.Path())
	err = os.WriteFile(configFile.Path()+".bak", orig, fileMode)
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	log.Infof(ctx, "Overriding %s", configFile.Path())
	return configFile.SaveTo(configFile.Path())
}
