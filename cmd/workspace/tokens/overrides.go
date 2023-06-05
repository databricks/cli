package tokens

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"ID"}}	{{"Expiry time"}}	{{"Comment"}}
	{{range .}}{{.TokenId|green}}	{{.ExpiryTime}}	{{.Comment}}
	{{end}}`)
}
