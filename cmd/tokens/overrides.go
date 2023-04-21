package tokens

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "ID"}}	{{white "Expiry time"}}	{{white "Comment"}}
	{{range .}}{{.TokenId|green}}	{{white "%d" .ExpiryTime}}	{{.Comment|white}}
	{{end}}`)
}
