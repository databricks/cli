package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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

	// Validate by probing the API surfaces this profile has a signal for.
	// Each signal — host shape or field presence — enables its corresponding
	// probe, and the OR of the probe results is the verdict.

	// Host signals.
	// isAccountHost:   classic accounts.* host.
	// isSPOGHost:      unified host with account-scoped OIDC discovery.
	// isWorkspaceHost: classic workspace host (neither of the above).
	isAccountHost := auth.IsClassicAccountHost(cfg.CanonicalHostName())
	isSPOGHost := auth.IsSPOGHost(cfg)
	isWorkspaceHost := auth.IsClassicWorkspaceHost(cfg)

	// Field signals.
	// hasAccountID:       account_id is set (from file, env, or discovery back-fill).
	// hasRealWorkspaceID: workspace_id is set to a real value.
	hasAccountID := cfg.AccountID != ""
	// workspace_id is "" when not present in the profile, "none" when the user picked Skip during SPOG login.
	hasRealWorkspaceID := cfg.WorkspaceID != "" && cfg.WorkspaceID != auth.WorkspaceIDNone

	tryAccount := isAccountHost || isSPOGHost || hasAccountID
	tryWorkspace := isWorkspaceHost || hasRealWorkspaceID

	var accountOK, workspaceOK bool
	if tryAccount {
		a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
		if err == nil {
			if _, err := a.Workspaces.List(ctx); err == nil {
				accountOK = true
			}
		}
	}
	if tryWorkspace {
		w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
		if err == nil {
			if _, err := w.CurrentUser.Me(ctx); err == nil {
				workspaceOK = true
			}
		}
	}

	// Capture Host/AuthType after the probes run: SDK Authenticate() sets
	// cfg.AuthType lazily based on the credentials it actually exercised.
	c.Host = cfg.Host
	c.AuthType = cfg.AuthType
	c.Valid = accountOK || workspaceOK
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
