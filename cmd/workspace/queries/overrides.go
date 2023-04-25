package queries

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	// TODO: figure out colored/non-colored headers and colspan shifts
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "ID"}}	{{white "Name"}}	{{white "Author"}}
	{{range .}}{{.Id|green}}	{{.Name|white}}	{{.User.Email|white}}
	{{end}}`)
}
