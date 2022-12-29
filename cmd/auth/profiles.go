package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/databricks/databricks-sdk-go/logger"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Lists profiles from ~/.databrickscfg",
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := os.Getenv("DATABRICKS_CONFIG_FILE")
		if configFile == "" {
			configFile = "~/.databrickscfg"
		}
		if strings.HasPrefix(configFile, "~") {
			homedir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot find homedir: %w", err)
			}
			configFile = fmt.Sprintf("%s%s", homedir, configFile[1:])
		}
		_, err := os.Stat(configFile)
		if os.IsNotExist(err) {
			// early return for non-configured machines
			logger.Debugf("%s not found on current host", configFile)
			return nil
		}
		iniFile, err := ini.Load(configFile)
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
