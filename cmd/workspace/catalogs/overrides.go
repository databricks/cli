package catalogs

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Type"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{blue "%s" .CatalogType}}	{{.Comment}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
