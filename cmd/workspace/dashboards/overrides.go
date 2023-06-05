package dashboards

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}
	{{range .}}{{.Id|green}}	{{.Name}}
	{{end}}`)
}
