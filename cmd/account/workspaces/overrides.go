package workspaces

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{green "%d" .WorkspaceId}}	{{.WorkspaceName}}	{{.WorkspaceStatus}}
	{{end}}`)
}
