package schemas

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"Full Name"}}	{{"Owner"}}	{{"Comment"}}
	{{range .}}{{.FullName|green}}	{{.Owner}}	{{.Comment}}
	{{end}}`)
}
