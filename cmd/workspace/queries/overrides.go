package queries

import (
	"github.com/databricks/cli/libs/cmdio"
	queries "github.com/databricks/cli/libs/sdktypes/queries/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *queries.ListQueriesRequest) {
	// TODO: figure out colored/non-colored headers and colspan shifts
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "Author"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.DisplayName|cyan}}	{{.OwnerUserName|cyan}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
