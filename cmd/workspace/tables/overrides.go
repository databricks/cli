package tables

import (
	"github.com/databricks/cli/libs/cmdio"
	uctables "github.com/databricks/cli/libs/sdktypes/uc/tables/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *uctables.ListTablesRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Table Type"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.FullName|green}}	{{blue "%s" .TableType}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
