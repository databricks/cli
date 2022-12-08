package dbfs

import (
	"github.com/databricks/bricks/cmd/dbfs/dbfs"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "dbfs",
	Short: `DBFS API makes it simple to interact with various data sources without having to include a users credentials every time to read a file.`,
	Long: `DBFS API makes it simple to interact with various data sources without having
  to include a users credentials every time to read a file.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(dbfs.Cmd)
}
