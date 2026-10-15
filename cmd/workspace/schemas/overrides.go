package schemas

import (
	"github.com/databricks/cli/libs/cmdio"
	ucschemas "github.com/databricks/cli/libs/sdktypes/uc/schemas/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *ucschemas.ListSchemasRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Owner"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.FullName|green}}	{{.Owner|cyan}}	{{.Comment}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
