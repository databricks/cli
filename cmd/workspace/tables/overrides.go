package tables

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *catalog.ListTablesRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Table Type"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.FullName|green}}	{{blue "%s" .TableType}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
