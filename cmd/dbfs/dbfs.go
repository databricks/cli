package dbfs

import (
	"github.com/databricks/bricks/cmd/dbfs/dbfs"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "dbfs",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(dbfs.Cmd)
}
