package instance_pools

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.InstancePoolId|green}}	{{.InstancePoolName}}	{{.NodeTypeId}}	{{.State}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
