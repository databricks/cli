package warehouses

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *sql.ListWarehousesRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "Size"}}	{{header "State"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.Name|cyan}}	{{.ClusterSize|cyan}}	{{if eq .State "RUNNING"}}{{"RUNNING"|green}}{{else if eq .State "STOPPED"}}{{"STOPPED"|red}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
