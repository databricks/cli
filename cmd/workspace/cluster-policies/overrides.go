package cluster_policies

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.PolicyId | green}}	{{.Name}}
	{{end}}`)

	getCmd.Annotations["template"] = cmdio.Heredoc(`{{.Definition | pretty_json}}`)
}
