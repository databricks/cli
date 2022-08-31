package configure

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

var host, token string

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure authentication",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		path := os.Getenv("DATABRICKS_CONFIG_FILE")
		if path == "" {
			path, err = os.UserHomeDir()
			if err != nil {
				panic(err)
			}
		}
		if filepath.Base(path) == ".databrickscfg" {
			path = filepath.Dir(path)
		}
		err = os.MkdirAll(path, os.ModeDir|os.ModePerm)
		if err != nil {
			panic(err)
		}
		cfgPath := filepath.Join(path, ".databrickscfg")
		_, err = os.Stat(cfgPath)
		if errors.Is(err, os.ErrNotExist) {
			file, err := os.Create(cfgPath)
			if err != nil {
				panic(err)
			}
			file.Close()
		} else if err != nil {
			panic(err)
		}
		cfg, err := ini.Load(cfgPath)
		if err != nil {
			panic(err)
		}

		cfg.Section("DEFAULT").Key("host").SetValue(host)
		cfg.Section("DEFAULT").Key("token").SetValue(token)

		var buffer bytes.Buffer
		buffer.WriteString("[DEFAULT]\n")
		_, err = cfg.WriteTo(&buffer)
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(cfgPath, buffer.Bytes(), os.ModePerm)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	root.RootCmd.AddCommand(configureCmd)
	configureCmd.Flags().StringVarP(&host, "host", "H", "", "Databricks host address")
	configureCmd.Flags().StringVarP(&token, "token", "t", "", "Databricks personal access token")
}
