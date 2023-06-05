package queries

import "github.com/databricks/cli/libs/cmdio"

func init() {
	// TODO: figure out colored/non-colored headers and colspan shifts
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "Author"}}
	{{range .}}{{.Id|green}}	{{.Name|cyan}}	{{.User.Email|cyan}}
	{{end}}`)
}
