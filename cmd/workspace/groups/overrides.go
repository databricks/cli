package groups

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	// Request only the columns the template shows unless --attributes is given.
	_ = listCmd.Flags().Lookup("attributes").Value.Set("id,displayName")
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.DisplayName}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
