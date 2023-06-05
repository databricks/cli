package queries

import "github.com/databricks/cli/libs/cmdio"

func init() {
	// TODO: figure out colored/non-colored headers and colspan shifts
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"ID"}}	{{"Name"}}	{{"Author"}}
	{{range .}}{{.Id|green}}	{{.Name}}	{{.User.Email}}
	{{end}}`)
}
