package clusters

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "ID"}}	{{white "Name"}}	{{white "State"}}
	{{range .}}{{.ClusterId | green}}	{{.ClusterName|white}}	{{if eq .State "RUNNING"}}{{green "%s" .State}}{{else if eq .State "TERMINATED"}}{{red "%s" .State}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)

	listNodeTypesCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .NodeTypes}}{{.NodeTypeId | green}}	{{.NumCores}}	{{.MemoryMb}}	{{.Category | blue}}
	{{end}}`)

	sparkVersionsCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .Versions}}{{.Key | green}}	{{.Name}}
	{{end}}
	`)
}
