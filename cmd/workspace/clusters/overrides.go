package clusters

import (
	"iter"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	clusters "github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"github.com/spf13/cobra"
)

// Below we add overrides for filter flags for cluster list command to allow for custom filtering
// Auto generating such flags is not yet supported by the CLI generator
func listOverride(listCmd *cobra.Command, listReq *clusters.ListClustersRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "State"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.ClusterId | green}}	{{.ClusterName | cyan}}	{{if eq .State "RUNNING"}}{{green "%s" .State}}{{else if eq .State "TERMINATED"}}{{red "%s" .State}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)

	var filter clusterFilter
	listCmd.Flags().BoolVar(&filter.isPinned, "is-pinned", false, "Filter clusters by pinned status")
	listCmd.Flags().StringVar(&filter.policyID, "policy-id", "", "Filter clusters by policy id")

	sources := &clusterSources{source: &filter.sources}
	listCmd.Flags().Var(sources, "cluster-sources", "Filter clusters by source")

	states := &clusterStates{state: &filter.states}
	listCmd.Flags().Var(states, "cluster-states", "Filter clusters by states")

	// The sdk-go list request has no filter_by, so filtered listings call the
	// REST API directly.
	runE := listCmd.RunE
	listCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !filter.isSet(cmd) {
			return runE(cmd, args)
		}
		ctx := cmd.Context()
		api, err := apiclient.New(cmdctx.WorkspaceClient(ctx).Config)
		if err != nil {
			return err
		}
		query := map[string]any{}
		if cmd.Flags().Changed("is-pinned") {
			query["filter_by.is_pinned"] = filter.isPinned
		}
		if filter.policyID != "" {
			query["filter_by.policy_id"] = filter.policyID
		}
		if len(filter.sources) > 0 {
			query["filter_by.cluster_sources"] = filter.sources
		}
		if len(filter.states) > 0 {
			query["filter_by.cluster_states"] = filter.states
		}
		if listReq.PageSize != nil {
			query["page_size"] = *listReq.PageSize
		}
		headers := auth.WorkspaceIDHeaders(api.Config)
		seq := func(yield func(*clusters.ClusterInfo, error) bool) {
			for {
				var resp clusters.ListClustersResponse
				if err := api.Do(ctx, "GET", "/api/2.1/clusters/list", headers, query, nil, &resp); err != nil {
					yield(nil, err)
					return
				}
				for i := range resp.Clusters {
					if !yield(&resp.Clusters[i], nil) {
						return
					}
				}
				if resp.NextPageToken == nil || *resp.NextPageToken == "" {
					return
				}
				query["page_token"] = *resp.NextPageToken
			}
		}
		return cmdio.RenderIterator(ctx, iter.Seq2[*clusters.ClusterInfo, error](seq))
	}
}

type clusterFilter struct {
	isPinned bool
	policyID string
	sources  []string
	states   []string
}

func (f *clusterFilter) isSet(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("is-pinned") || f.policyID != "" || len(f.sources) > 0 || len(f.states) > 0
}

type clusterSources struct {
	source *[]string
}

func (c *clusterSources) String() string {
	s := make([]string, len(*c.source))
	for i, source := range *c.source {
		s[i] = source
	}

	return strings.Join(s, ",")
}

func (c *clusterSources) Set(value string) error {
	splits := strings.SplitSeq(value, ",")
	for split := range splits {
		*c.source = append(*c.source, split)
	}

	return nil
}

func (c *clusterSources) Type() string {
	return "[]string"
}

type clusterStates struct {
	state *[]string
}

func (c *clusterStates) String() string {
	s := make([]string, len(*c.state))
	for i, source := range *c.state {
		s[i] = source
	}

	return strings.Join(s, ",")
}

func (c *clusterStates) Set(value string) error {
	splits := strings.SplitSeq(value, ",")
	for split := range splits {
		*c.state = append(*c.state, split)
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
