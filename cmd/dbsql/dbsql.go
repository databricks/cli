package dbsql

import (
	"github.com/databricks/bricks/cmd/dbsql/alerts"
	"github.com/databricks/bricks/cmd/dbsql/dashboards"
	data_sources "github.com/databricks/bricks/cmd/dbsql/data-sources"
	dbsql_permissions "github.com/databricks/bricks/cmd/dbsql/dbsql-permissions"
	"github.com/databricks/bricks/cmd/dbsql/queries"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "dbsql",
}

func init() {

	Cmd.AddCommand(alerts.Cmd)
	Cmd.AddCommand(dashboards.Cmd)
	Cmd.AddCommand(data_sources.Cmd)
	Cmd.AddCommand(dbsql_permissions.Cmd)
	Cmd.AddCommand(queries.Cmd)
}
