package dashboards

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "ID"}}	{{white "Name"}}
	{{range .}}{{.Id|green}}	{{.Name}}
	{{end}}`)
}
