package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

type profileMetadata struct {
	Name        string `json:"name"`
	Host        string `json:"host,omitempty"`
	AccountID   string `json:"account_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Cloud       string `json:"cloud"`
	AuthType    string `json:"auth_type"`
	Valid       bool   `json:"valid"`
	Default     bool   `json:"default,omitempty"`
}

func (c *profileMetadata) IsEmpty() bool {
	return c.Host == "" && c.AccountID == ""
}

func (c *profileMetadata) Load(ctx context.Context, configFilePath string, skipValidate bool) {
	cfg := &config.Config{
		Loaders:           []config.Loader{config.ConfigFile},
		ConfigFile:        configFilePath,
		Profile:           c.Name,
		DatabricksCliPath: env.Get(ctx, "DATABRICKS_CLI_PATH"),
	}
	_ = cfg.EnsureResolved()
	if cfg.IsAws() {
		c.Cloud = "aws"
	} else if cfg.IsAzure() {
		c.Cloud = "azure"
	} else if cfg.IsGcp() {
		c.Cloud = "gcp"
	}

	if skipValidate {
		c.Host = cfg.CanonicalHostName()
		c.AuthType = cfg.AuthType
		return
	}

	// ConfigType() classifies based on the host URL prefix (accounts.* →
	// AccountConfig, everything else → WorkspaceConfig). SPOG hosts don't
	// match the accounts.* prefix so they're misclassified as WorkspaceConfig.
	// Use the resolved DiscoveryURL (from .well-known/databricks-config) to
	// detect SPOG hosts with account-scoped OIDC, matching the routing logic
	// in auth.AuthArguments.ToOAuthArgument().
	configType := cfg.ConfigType()
	isAccountScopedOIDC := cfg.DiscoveryURL != "" && strings.Contains(cfg.DiscoveryURL, "/oidc/accounts/")
	if configType != config.AccountConfig && cfg.AccountID != "" && isAccountScopedOIDC {
		if cfg.WorkspaceID != "" && cfg.WorkspaceID != auth.WorkspaceIDNone {
			configType = config.WorkspaceConfig
		} else {
			configType = config.AccountConfig
		}
	}

	// Legacy backward compat: profiles with Experimental_IsUnifiedHost where
	// .well-known is unreachable (so DiscoveryURL is empty). Matches the
	// fallback in auth.AuthArguments.ToOAuthArgument().
	if configType == config.InvalidConfig && cfg.Experimental_IsUnifiedHost && cfg.AccountID != "" {
		configType = config.AccountConfig
	}

	switch configType {
	case config.AccountConfig:
		a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
		if err != nil {
			return
		}
		_, err = a.Workspaces.List(ctx)
		c.Host = cfg.Host
		c.AuthType = cfg.AuthType
		if err != nil {
			return
		}
		c.Valid = true
	case config.WorkspaceConfig:
		w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
		if err != nil {
			return
		}
		_, err = w.CurrentUser.Me(ctx)
		c.Host = cfg.Host
		c.AuthType = cfg.AuthType
		if err != nil {
			return
		}
		c.Valid = true
	case config.InvalidConfig:
		// Invalid configuration, skip validation
		return
	}
}

func newProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Lists profiles from ~/.databrickscfg",
		Annotations: map[string]string{
			"template": cmdio.Heredoc(`
			{{header "Name"}}	{{header "Host"}}	{{header "Valid"}}
			{{range .Profiles}}{{.Name | green}}{{if .Default}} (Default){{end}}	{{.Host|cyan}}	{{bool .Valid}}
			{{end}}`),
		},
	}

	var skipValidate bool
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Whether to skip validating the profiles")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var profiles []*profileMetadata
		iniFile, err := profile.DefaultProfiler.Get(cmd.Context())
		if errors.Is(err, fs.ErrNotExist) {
			// return empty list for non-configured machines
			iniFile = &config.File{
				File: &ini.File{},
			}
		} else if err != nil {
			return fmt.Errorf("cannot parse config file: %w", err)
		}

		defaultProfile := databrickscfg.GetConfiguredDefaultProfileFrom(iniFile)

		var wg sync.WaitGroup
		for _, v := range iniFile.Sections() {
			hash := v.KeysHash()
			profile := &profileMetadata{
				Name:        v.Name(),
				Host:        hash["host"],
				AccountID:   hash["account_id"],
				WorkspaceID: hash["workspace_id"],
				Default:     v.Name() == defaultProfile,
			}
			if profile.IsEmpty() {
				continue
			}
			wg.Go(func() {
				ctx := cmd.Context()
				t := time.Now()
				profile.Load(ctx, iniFile.Path(), skipValidate)
				log.Debugf(ctx, "Profile %q took %s to load", profile.Name, time.Since(t))
			})
			profiles = append(profiles, profile)
		}
		wg.Wait()
		return cmdio.Render(cmd.Context(), struct {
			Profiles []*profileMetadata `json:"profiles"`
		}{profiles})
	}

	return cmd
}
