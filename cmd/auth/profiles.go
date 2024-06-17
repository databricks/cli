package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
	Name      string `json:"name"`
	Host      string `json:"host,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Cloud     string `json:"cloud"`
	AuthType  string `json:"auth_type"`
	Valid     bool   `json:"valid"`
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

	if skipValidate {
		c.Host = cfg.CanonicalHostName()
		c.AuthType = cfg.AuthType
		return
	}

	if cfg.IsAccountClient() {
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
	} else {
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
	}
}

func newProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Lists profiles from ~/.databrickscfg",
		Annotations: map[string]string{
			"template": cmdio.Heredoc(`
			{{header "Name"}}	{{header "Host"}}	{{header "Valid"}}
			{{range .Profiles}}{{.Name | green}}	{{.Host|cyan}}	{{bool .Valid}}
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
			wg.Add(1)
			go func() {
				ctx := cmd.Context()
				t := time.Now()
				profile.Load(ctx, iniFile.Path(), skipValidate)
				log.Debugf(ctx, "Profile %q took %s to load", profile.Name, time.Since(t))
				wg.Done()
			}()
			profiles = append(profiles, profile)
		}
		wg.Wait()
		return cmdio.Render(cmd.Context(), struct {
			Profiles []*profileMetadata `json:"profiles"`
		}{profiles})
	}

	return cmd
}
