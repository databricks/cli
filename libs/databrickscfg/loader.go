package databrickscfg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/bricks/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"gopkg.in/ini.v1"
)

var ResolveProfileFromHost = profileFromHostLoader{}

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

	configFile, err := LoadFile(cfg.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cannot parse config file: %w", err)
	}

	// Normalized version of the configured host.
	host := normalizeHost(cfg.Host)

	// Look for sections in the configuration file that match the configured host.
	var matching []*ini.Section
	for _, section := range configFile.Sections() {
		key, err := section.GetKey("host")
		if err != nil {
			log.Tracef(context.Background(), "section %s: %s", section.Name(), err)
			continue
		}

		// Ignore this section if the normalized host doesn't match.
		if normalizeHost(key.Value()) != host {
			continue
		}

		matching = append(matching, section)
	}

	// If there are no matching sections, we don't do anything.
	if len(matching) == 0 {
		return nil
	}

	// If there are multiple matching sections, let the user know it is impossible
	// to unambiguously select a profile to use.
	if len(matching) > 1 {
		var names []string
		for _, section := range matching {
			names = append(names, section.Name())
		}

		return fmt.Errorf(
			"multiple profiles for host %s (%s): please set DATABRICKS_CONFIG_PROFILE to specify one",
			host,
			strings.Join(names, ", "),
		)
	}

	match := matching[0]
	log.Debugf(context.Background(), "Loading profile %s because of host match", match.Name())
	err = config.ConfigAttributes.ResolveFromStringMap(cfg, match.KeysHash())
	if err != nil {
		return fmt.Errorf("%s %s profile: %w", configFile.Path(), match.Name(), err)
	}

	return nil

}

func (l profileFromHostLoader) isAnyAuthConfigured(cfg *config.Config) bool {
	for _, a := range config.ConfigAttributes {
		if a.Auth == "" {
			continue
		}
		if !a.IsZero(cfg) {
			return true
		}
	}
	return false
}
