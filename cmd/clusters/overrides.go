package clusters

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.ClusterId | green}}	{{.ClusterName}}	{{.State}}
	{{end}}`)

	listNodeTypesCmd.Annotations["template"] = ui.Heredoc(`
	{{range .NodeTypes}}{{.NodeTypeId | green}}	{{.NumCores}}	{{.MemoryMb}}	{{.Category | blue}}
	{{end}}`)

	sparkVersionsCmd.Annotations["template"] = ui.Heredoc(`
	{{range .Versions}}{{.Key | green}}	{{.Name}}
	{{end}}
	`)
}
