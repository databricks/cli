package dashboards

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "ID"}}	{{white "Name"}}
	{{range .}}{{.Id|green}}	{{.Name}}
	{{end}}`)
}
