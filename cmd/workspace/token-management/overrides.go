package token_management

import (
	"github.com/databricks/cli/libs/cmdio"
	tokenmanagement "github.com/databricks/cli/libs/sdktypes/tokenmanagement/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *tokenmanagement.ListTokensRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Created By"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.TokenId|green}}	{{.CreatedByUsername|cyan}}	{{.Comment|cyan}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
