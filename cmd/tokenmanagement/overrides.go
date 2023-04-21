package tokenmanagement

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "ID"}}	{{white "Created By"}}	{{white "Comment"}}
	{{range .}}{{.TokenId|green}}	{{.CreatedByUsername|white}}	{{.Comment|white}}
	{{end}}`)
}
