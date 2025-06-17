package metastores

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, req *catalog.ListMetastoresRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{"Region"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.MetastoreId|green}}	{{.Name|cyan}}	{{.Region}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
