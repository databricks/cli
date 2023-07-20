package networks

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.NetworkId | green}}	{{.NetworkName}}	{{.WorkspaceId}}	{{.VpcStatus}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
