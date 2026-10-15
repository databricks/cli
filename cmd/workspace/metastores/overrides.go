package metastores

import (
	"github.com/databricks/cli/libs/cmdio"
	ucmetastores "github.com/databricks/cli/libs/sdktypes/uc/metastores/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, req *ucmetastores.ListMetastoresRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{"Region"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.MetastoreId|green}}	{{.Name|cyan}}	{{.Region}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
