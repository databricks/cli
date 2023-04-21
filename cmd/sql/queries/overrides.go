package queries

import "github.com/databricks/bricks/lib/ui"

func init() {
	// TODO: figure out colored/non-colored headers and colspan shifts
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "ID"}}	{{white "Name"}}	{{white "Author"}}
	{{range .}}{{.Id|green}}	{{.Name|white}}	{{.User.Email|white}}
	{{end}}`)
}
