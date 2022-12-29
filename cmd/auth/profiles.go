package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/databricks/databricks-sdk-go/logger"
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
		configFile = fmt.Sprintf("%s%s", homedir, configFile[1:])
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
			logger.Debugf("Databricks config not found on current host")
			return nil
		}
		if err != nil {
			return fmt.Errorf("cannot parse config file: %w", err)
		}
		w := cmd.OutOrStdout()
		for _, v := range iniFile.Sections() {
			hash := v.KeysHash()
			host := hash["host"]
			if host == "" {
				host = hash["azure_workspace_resource_id"]
			}
			fmt.Fprintf(w, "%s\t%s\n", v.Name(), host)
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(profilesCmd)
}
