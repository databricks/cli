package schemas

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *catalog.ListSchemasRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Owner"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.FullName|green}}	{{.Owner|cyan}}	{{.Comment}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
