package tokens

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Expiry time"}}	{{header "Comment"}}
	{{range .}}{{.TokenId|green}}	{{cyan "%d" .ExpiryTime}}	{{.Comment|cyan}}
	{{end}}`)
}
