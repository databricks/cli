package sql

import (
	"github.com/databricks/bricks/cmd/sql/alerts"
	"github.com/databricks/bricks/cmd/sql/dashboards"
	data_sources "github.com/databricks/bricks/cmd/sql/data-sources"
	dbsql_permissions "github.com/databricks/bricks/cmd/sql/dbsql-permissions"
	"github.com/databricks/bricks/cmd/sql/queries"
	query_history "github.com/databricks/bricks/cmd/sql/query-history"
	"github.com/databricks/bricks/cmd/sql/warehouses"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "sql",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(alerts.Cmd)
	Cmd.AddCommand(dashboards.Cmd)
	Cmd.AddCommand(data_sources.Cmd)
	Cmd.AddCommand(dbsql_permissions.Cmd)
	Cmd.AddCommand(queries.Cmd)
	Cmd.AddCommand(query_history.Cmd)
	Cmd.AddCommand(warehouses.Cmd)
}
