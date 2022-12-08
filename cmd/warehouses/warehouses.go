package warehouses

import (
	query_history "github.com/databricks/bricks/cmd/warehouses/query-history"
	"github.com/databricks/bricks/cmd/warehouses/warehouses"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "warehouses",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(query_history.Cmd)
	Cmd.AddCommand(warehouses.Cmd)
}
