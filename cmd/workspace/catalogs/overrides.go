package catalogs

import (
	"github.com/databricks/cli/libs/cmdio"
	uccatalogs "github.com/databricks/cli/libs/sdktypes/uc/catalogs/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *uccatalogs.ListCatalogsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Type"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{blue "%s" .CatalogType}}	{{.Comment}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
