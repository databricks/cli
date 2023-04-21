package groups

import "github.com/databricks/bricks/lib/ui"

func init() {
	listReq.Attributes = "id,displayName"
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.Id|green}}	{{.DisplayName}}
	{{end}}`)
}
