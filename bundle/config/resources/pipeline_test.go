package resources

import (
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineMergeClusters(t *testing.T) {
	p := &Pipeline{
		PipelineSpec: &pipelines.PipelineSpec{
			Clusters: []pipelines.PipelineCluster{
				{
					NodeTypeId: "i3.xlarge",
					NumWorkers: 2,
					PolicyId:   "1234",
				},
				{
					Label:      "maintenance",
					NodeTypeId: "i3.2xlarge",
				},
				{
					NodeTypeId: "i3.2xlarge",
					NumWorkers: 4,
				},
			},
		},
	}

	err := p.MergeClusters()
	require.NoError(t, err)

	assert.Len(t, p.Clusters, 2)
	assert.Equal(t, "default", p.Clusters[0].Label)
	assert.Equal(t, "maintenance", p.Clusters[1].Label)

	// The default cluster was merged with a subsequent one.
	pc0 := p.Clusters[0]
	assert.Equal(t, "i3.2xlarge", pc0.NodeTypeId)
	assert.Equal(t, 4, pc0.NumWorkers)
	assert.Equal(t, "1234", pc0.PolicyId)

	// The maintenance cluster was left untouched.
	pc1 := p.Clusters[1]
	assert.Equal(t, "i3.2xlarge", pc1.NodeTypeId)
}

func TestPipelineMergeClustersCaseInsensitive(t *testing.T) {
	p := &Pipeline{
		PipelineSpec: &pipelines.PipelineSpec{
			Clusters: []pipelines.PipelineCluster{
				{
					Label:      "default",
					NumWorkers: 2,
				},
				{
					Label:      "DEFAULT",
					NumWorkers: 4,
				},
			},
		},
	}

	err := p.MergeClusters()
	require.NoError(t, err)

	assert.Len(t, p.Clusters, 1)

	// The default cluster was merged with a subsequent one.
	pc0 := p.Clusters[0]
	assert.Equal(t, "default", strings.ToLower(pc0.Label))
	assert.Equal(t, 4, pc0.NumWorkers)
}
