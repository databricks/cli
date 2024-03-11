package dashboards

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *sql.ListDashboardsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.Name}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
