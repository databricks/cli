package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		type configProfile struct {
			Name            string `json:"name"`
			Host            string `json:"host,omitempty"`
			AzureResourceID string `json:"azure_workspace_resource_id,omitempty"`
			AccountID       string `json:"account_id,omitempty"`
		}
		var profiles []configProfile
		for _, v := range iniFile.Sections() {
			hash := v.KeysHash()
			profiles = append(profiles, configProfile{
				Name:            v.Name(),
				Host:            hash["host"],
				AzureResourceID: hash["azure_workspace_resource_id"],
				AccountID:       hash["account_id"],
			})
		}
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
