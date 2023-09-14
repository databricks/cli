package token_management

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *settings.ListTokenManagementRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Created By"}}	{{header "Comment"}}
	{{range .}}{{.TokenId|green}}	{{.CreatedByUsername|cyan}}	{{.Comment|cyan}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
