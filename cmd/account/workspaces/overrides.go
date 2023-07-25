package workspaces

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .WorkspaceId}}	{{.WorkspaceName}}	{{.WorkspaceStatus}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
