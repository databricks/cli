package token_management

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"ID"}}	{{"Created By"}}	{{"Comment"}}
	{{range .}}{{.TokenId|green}}	{{.CreatedByUsername}}	{{.Comment}}
	{{end}}`)
}
