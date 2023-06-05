package dashboards

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"ID"}}	{{"Name"}}
	{{range .}}{{.Id|green}}	{{.Name}}
	{{end}}`)
}
