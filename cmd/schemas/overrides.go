package schemas

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "Full Name"}}	{{white "Owner"}}	{{white "Comment"}}
	{{range .}}{{.FullName|green}}	{{.Owner|white}}	{{.Comment}}
	{{end}}`)
}
