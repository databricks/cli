package vector_search_endpoints

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/spf13/cobra"
)

// createEndpointSkeletonOverride adds --generate-skeleton to
// `vector-search-endpoints create-endpoint`, reusing the shared helper.
func createEndpointSkeletonOverride(cmd *cobra.Command, createEndpointReq *vectorsearch.CreateEndpoint) {
	root.RegisterGenerateSkeleton(cmd, createEndpointReq)
}

func init() {
	createEndpointOverrides = append(createEndpointOverrides, createEndpointSkeletonOverride)
}
