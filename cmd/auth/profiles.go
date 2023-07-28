package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

func getDatabricksCfg() (*ini.File, error) {
	configFile := os.Getenv("DATABRICKS_CONFIG_FILE")
	if configFile == "" {
		configFile = "~/.databrickscfg"
	}
	if strings.HasPrefix(configFile, "~") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot find homedir: %w", err)
		}
		configFile = filepath.Join(homedir, configFile[1:])
	}
	return ini.Load(configFile)
}

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

func (c *profileMetadata) Load(ctx context.Context, skipValidate bool) {
	// TODO: disable config loaders other than configfile
	cfg := &config.Config{Profile: c.Name}
	_ = cfg.EnsureResolved()
	if cfg.IsAws() {
		c.Cloud = "aws"
	} else if cfg.IsAzure() {
		c.Cloud = "azure"
	} else if cfg.IsGcp() {
		c.Cloud = "gcp"
	}

	if skipValidate {
		err := cfg.Authenticate(&http.Request{
			Header: make(http.Header),
		})
		if err != nil {
			return
		}
		c.AuthType = cfg.AuthType
		return
	}

	if cfg.IsAccountClient() {
		a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
		if err != nil {
			return
		}
		_, err = a.Workspaces.List(ctx)
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
		c.AuthType = cfg.AuthType
		if err != nil {
			return
		}
		c.Valid = true
	}
	// set host again, this time normalized
	c.Host = cfg.Host
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
		iniFile, err := getDatabricksCfg()
		if os.IsNotExist(err) {
			// return empty list for non-configured machines
			iniFile = ini.Empty()
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
				// load more information about profile
				profile.Load(cmd.Context(), skipValidate)
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
