package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

func (c *profileMetadata) Load(ctx context.Context) {
	cfg := &config.Config{Profile: c.Name}
	_ = cfg.EnsureResolved()
	if cfg.IsAws() {
		c.Cloud = "aws"
	} else if cfg.IsAzure() {
		c.Cloud = "azure"
	} else if cfg.IsGcp() {
		c.Cloud = "gcp"
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
		_, err = w.Tokens.ListAll(ctx)
		c.AuthType = cfg.AuthType
		if err != nil {
			return
		}
		c.Valid = true
	}
	// set host again, this time normalized
	c.Host = cfg.Host
}

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Lists profiles from ~/.databrickscfg",
	RunE: func(cmd *cobra.Command, args []string) error {
		iniFile, err := getDatabricksCfg()
		if os.IsNotExist(err) {
			// early return for non-configured machines
			return errors.New("~/.databrickcfg not found on current host")
		}
		if err != nil {
			return fmt.Errorf("cannot parse config file: %w", err)
		}
		var profiles []*profileMetadata
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
				profile.Load(cmd.Context())
				wg.Done()
			}()
			profiles = append(profiles, profile)
		}
		wg.Wait()
		raw, err := json.MarshalIndent(map[string]any{
			"profiles": profiles,
		}, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(raw)
		return nil
	},
}

func init() {
	authCmd.AddCommand(profilesCmd)
}
