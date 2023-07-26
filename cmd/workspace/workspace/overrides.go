package workspace

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *workspace.ListWorkspaceRequest) {
	listReq.Path = "/"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Type"}}	{{header "Language"}}	{{header "Path"}}
	{{range .}}{{green "%d" .ObjectId}}	{{blue "%s" .ObjectType}}	{{cyan "%s" .Language}}	{{.Path|cyan}}
	{{end}}`)
}

func exportOverride(exportCmd *cobra.Command, exportReq *workspace.ExportRequest) {
	// The export command prints the contents of the file to stdout by default.
	exportCmd.Annotations["template"] = `{{.Content | b64_decode}}`
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	exportOverrides = append(exportOverrides, exportOverride)
}
