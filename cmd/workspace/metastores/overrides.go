package metastores

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{"Region"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.MetastoreId|green}}	{{.Name|cyan}}	{{.Region}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
