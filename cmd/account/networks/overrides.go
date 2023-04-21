package networks

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.NetworkId | green}}	{{.NetworkName}}	{{.WorkspaceId}}	{{.VpcStatus}}
	{{end}}`)
}
