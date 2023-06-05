package token_management

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Created By"}}	{{header "Comment"}}
	{{range .}}{{.TokenId|green}}	{{.CreatedByUsername|cyan}}	{{.Comment|cyan}}
	{{end}}`)
}
