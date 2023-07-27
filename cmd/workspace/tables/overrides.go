package tables

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *catalog.ListTablesRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Table Type"}}
	{{range .}}{{.FullName|green}}	{{blue "%s" .TableType}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
