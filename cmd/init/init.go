package init

import (
	"encoding/json"
	"os"

	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

const SchemaFileName = "config.json"

// root template defination at schema.json
// decide on semantics of defination later

// initCmd represents the fs command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Template",
	Long:  `Initialize bundle template`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var config map[string]interface{}
		b, err := os.ReadFile(SchemaFileName)
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &config)
		if err != nil {
			return err
		}
		err = walkFileTree(config, ".", ".")
		if err != nil {
			err2 := os.RemoveAll("favela")
			if err2 != nil {
				return err2
			}
			return err
		}
		return nil
	},
}

func init() {
	root.RootCmd.AddCommand(initCmd)
}
