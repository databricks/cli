package warehouses

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *sql.ListWarehousesRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "Size"}}	{{header "State"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.Name|cyan}}	{{.ClusterSize|cyan}}	{{if eq .State "RUNNING"}}{{"RUNNING"|green}}{{else if eq .State "STOPPED"}}{{"STOPPED"|red}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "ID", Extract: func(v any) string {
			return v.(sql.EndpointInfo).Id
		}},
		{Header: "Name", Extract: func(v any) string {
			return v.(sql.EndpointInfo).Name
		}},
		{Header: "Size", Extract: func(v any) string {
			return v.(sql.EndpointInfo).ClusterSize
		}},
		{Header: "State", Extract: func(v any) string {
			return string(v.(sql.EndpointInfo).State)
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
