package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

type profileMetadata struct {
	Name              string `json:"name"`
	Host              string `json:"host,omitempty"`
	AccountID         string `json:"account_id,omitempty"`
	Cloud             string `json:"cloud"`
	AuthType          string `json:"auth_type"`
	Scopes            string `json:"scopes,omitempty"`
	ClientID          string `json:"client_id,omitempty"`
	Valid             bool   `json:"valid"`
	ValidationSkipped bool   `json:"validation_skipped,omitempty"`
}

func (c *profileMetadata) IsEmpty() bool {
	return c.Host == "" && c.AccountID == ""
}

func (c *profileMetadata) Load(ctx context.Context, configFilePath string, skipValidate bool) {
	cfg := &config.Config{
		Loaders:    []config.Loader{config.ConfigFile},
		ConfigFile: configFilePath,
		Profile:    c.Name,
	}
	_ = cfg.EnsureResolved()
	if cfg.IsAws() {
		c.Cloud = "aws"
	} else if cfg.IsAzure() {
		c.Cloud = "azure"
	} else if cfg.IsGcp() {
		c.Cloud = "gcp"
	}

	c.Scopes = strings.Join(cfg.GetScopes(), ",")
	c.ClientID = cfg.ClientID

	// Check if all-apis scope is present. If not, validation may be unreliable
	// because the validation API calls may not be accessible with restricted scopes.
	hasAllApisScope := slices.Contains(cfg.GetScopes(), "all-apis")

	if skipValidate {
		c.Host = cfg.CanonicalHostName()
		c.AuthType = cfg.AuthType
		return
	}

	if !hasAllApisScope {
		c.Host = cfg.CanonicalHostName()
		c.AuthType = cfg.AuthType
		c.ValidationSkipped = true
		return
	}

	switch cfg.ConfigType() {
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
			{{header "Name"}}	{{header "Host"}}	{{header "Client ID"}}	{{header "Scopes"}}	{{header "Valid"}}
			{{range .}}{{.Name | green}}	{{.Host | cyan}}	{{if .ClientID}}{{.ClientID | magenta}}{{else}}{{ "-" | magenta}}{{end}}	{{if .Scopes}}{{.Scopes | yellow}}{{else}}{{"all-apis" | yellow}}{{end}}	{{if .ValidationSkipped}}{{ "-" | yellow}}{{else}}{{bool .Valid}}{{end}}
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
		var wg sync.WaitGroup
		for _, v := range iniFile.Sections() {
			hash := v.KeysHash()
			profile := &profileMetadata{
				Name:      v.Name(),
				Host:      hash["host"],
				AccountID: hash["account_id"],
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
		err = cmdio.Render(cmd.Context(), profiles)
		if err != nil {
			return err
		}

		for _, p := range profiles {
			if p.ValidationSkipped {
				cmdio.LogString(cmd.Context(),
					"\nNote: Validation is skipped for profiles without the 'all-apis' scope "+
						"because the validation API may not be accessible with restricted scopes.")
				break
			}
		}

		return nil
	}

	return cmd
}
