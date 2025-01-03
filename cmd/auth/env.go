package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

func canonicalHost(host string) (string, error) {
	parsedHost, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	// If the host is empty, assume the scheme wasn't included.
	if parsedHost.Host == "" {
		return "https://" + host, nil
	}
	return "https://" + parsedHost.Host, nil
}

var ErrNoMatchingProfiles = errors.New("no matching profiles found")

func resolveSection(cfg *config.Config, iniFile *config.File) (*ini.Section, error) {
	var candidates []*ini.Section
	configuredHost, err := canonicalHost(cfg.Host)
	if err != nil {
		return nil, err
	}
	for _, section := range iniFile.Sections() {
		hash := section.KeysHash()
		host, ok := hash["host"]
		if !ok {
			// if host is not set
			continue
		}
		canonical, err := canonicalHost(host)
		if err != nil {
			// we're fine with other corrupt profiles
			continue
		}
		if canonical != configuredHost {
			continue
		}
		candidates = append(candidates, section)
	}
	if len(candidates) == 0 {
		return nil, ErrNoMatchingProfiles
	}
	// in the real situations, we don't expect this to happen often
	// (if not at all), hence we don't trim the list
	if len(candidates) > 1 {
		var profiles []string
		for _, v := range candidates {
			profiles = append(profiles, v.Name())
		}
		return nil, fmt.Errorf("%s match %s in %s",
			strings.Join(profiles, " and "), cfg.Host, cfg.ConfigFile)
	}
	return candidates[0], nil
}

func loadFromDatabricksCfg(ctx context.Context, cfg *config.Config) error {
	iniFile, err := profile.DefaultProfiler.Get(ctx)
	if errors.Is(err, fs.ErrNotExist) {
		// it's fine not to have ~/.databrickscfg
		return nil
	}
	if err != nil {
		return err
	}
	profile, err := resolveSection(cfg, iniFile)
	if err == ErrNoMatchingProfiles {
		// it's also fine for Azure CLI or Databricks CLI, which
		// are resolved by unified auth handling in the Go SDK.
		return nil
	}
	if err != nil {
		return err
	}
	cfg.Profile = profile.Name()
	return nil
}

func newEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Get env",
	}

	var host string
	var profile string
	cmd.Flags().StringVar(&host, "host", host, "Hostname to get auth env for")
	cmd.Flags().StringVar(&profile, "profile", profile, "Profile to get auth env for")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cfg := &config.Config{
			Host:    host,
			Profile: profile,
		}
		if profile != "" {
			cfg.Profile = profile
		} else if cfg.Host == "" {
			cfg.Profile = "DEFAULT"
		} else if err := loadFromDatabricksCfg(cmd.Context(), cfg); err != nil {
			return err
		}
		// Go SDK is lazy loaded because of Terraform semantics,
		// so we're creating a dummy HTTP request as a placeholder
		// for headers.
		r := &http.Request{Header: http.Header{}}
		err := cfg.Authenticate(r.WithContext(cmd.Context()))
		if err != nil {
			return err
		}
		vars := map[string]string{}
		for _, a := range config.ConfigAttributes {
			if a.IsZero(cfg) {
				continue
			}
			envValue := a.GetString(cfg)
			for _, envName := range a.EnvVars {
				vars[envName] = envValue
			}
		}
		raw, err := json.MarshalIndent(map[string]any{
			"env": vars,
		}, "", "  ")
		if err != nil {
			return err
		}
		_, _ = cmd.OutOrStdout().Write(raw)
		return nil
	}

	return cmd
}
