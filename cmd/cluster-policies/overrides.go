package cluster_policies

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.PolicyId | green}}	{{.Name}}
	{{end}}`)

	getCmd.Annotations["template"] = ui.Heredoc(`{{.Definition | pretty_json}}`)
}
