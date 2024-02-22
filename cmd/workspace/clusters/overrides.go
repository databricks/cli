package clusters

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *compute.ListClustersRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "State"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.ClusterId | green}}	{{.ClusterName | cyan}}	{{if eq .State "RUNNING"}}{{green "%s" .State}}{{else if eq .State "TERMINATED"}}{{red "%s" .State}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)
}

func listNodeTypesOverride(listNodeTypesCmd *cobra.Command) {
	listNodeTypesCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .NodeTypes}}{{.NodeTypeId | green}}	{{.NumCores}}	{{.MemoryMb}}	{{.Category | blue}}
	{{end}}`)
}

func sparkVersionsOverride(sparkVersionsCmd *cobra.Command) {
	sparkVersionsCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .Versions}}{{.Key | green}}	{{.Name}}
	{{end}}
	`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	listNodeTypesOverrides = append(listNodeTypesOverrides, listNodeTypesOverride)
	sparkVersionsOverrides = append(sparkVersionsOverrides, sparkVersionsOverride)
}
