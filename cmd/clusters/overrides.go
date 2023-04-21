package clusters

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "ID"}}	{{white "Name"}}	{{white "State"}}
	{{range .}}{{.ClusterId | green}}	{{.ClusterName|white}}	{{if eq .State "RUNNING"}}{{green "%s" .State}}{{else if eq .State "TERMINATED"}}{{red "%s" .State}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)

	listNodeTypesCmd.Annotations["template"] = ui.Heredoc(`
	{{range .NodeTypes}}{{.NodeTypeId | green}}	{{.NumCores}}	{{.MemoryMb}}	{{.Category | blue}}
	{{end}}`)

	sparkVersionsCmd.Annotations["template"] = ui.Heredoc(`
	{{range .Versions}}{{.Key | green}}	{{.Name}}
	{{end}}
	`)
}
