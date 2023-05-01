package instance_pools

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.InstancePoolId|green}}	{{.InstancePoolName}}	{{.NodeTypeId}}	{{.State}}
	{{end}}`)
}
