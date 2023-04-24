package token_management

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "ID"}}	{{white "Created By"}}	{{white "Comment"}}
	{{range .}}{{.TokenId|green}}	{{.CreatedByUsername|white}}	{{.Comment|white}}
	{{end}}`)
}
