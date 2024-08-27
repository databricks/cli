package clusters

import (
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

// Below we add overrides for filter flags for cluster list command to allow for custom filtering
// Auto generating such flags is not yet supported by the CLI generator
func listOverride(listCmd *cobra.Command, listReq *compute.ListClustersRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "State"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.ClusterId | green}}	{{.ClusterName | cyan}}	{{if eq .State "RUNNING"}}{{green "%s" .State}}{{else if eq .State "TERMINATED"}}{{red "%s" .State}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)

	listReq.FilterBy = &compute.ListClustersFilterBy{}
	listCmd.Flags().BoolVar(&listReq.FilterBy.IsPinned, "is-pinned", false, "Filter clusters by pinned status")
	listCmd.Flags().StringVar(&listReq.FilterBy.PolicyId, "policy-id", "", "Filter clusters by policy id")

	sources := &clusterSources{source: &listReq.FilterBy.ClusterSources}
	listCmd.Flags().Var(sources, "cluster-sources", "Filter clusters by source")

	states := &clusterStates{state: &listReq.FilterBy.ClusterStates}
	listCmd.Flags().Var(states, "cluster-states", "Filter clusters by states")
}

type clusterSources struct {
	source *[]compute.ClusterSource
}

func (c *clusterSources) String() string {
	s := make([]string, len(*c.source))
	for i, source := range *c.source {
		s[i] = string(source)
	}

	return strings.Join(s, ",")
}

func (c *clusterSources) Set(value string) error {
	splits := strings.Split(value, ",")
	for _, split := range splits {
		*c.source = append(*c.source, compute.ClusterSource(split))
	}

	return nil
}

func (c *clusterSources) Type() string {
	return "[]string"
}

type clusterStates struct {
	state *[]compute.State
}

func (c *clusterStates) String() string {
	s := make([]string, len(*c.state))
	for i, source := range *c.state {
		s[i] = string(source)
	}

	return strings.Join(s, ",")
}

func (c *clusterStates) Set(value string) error {
	splits := strings.Split(value, ",")
	for _, split := range splits {
		*c.state = append(*c.state, compute.State(split))
	}

	return nil
}

func (c *clusterStates) Type() string {
	return "[]string"
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
