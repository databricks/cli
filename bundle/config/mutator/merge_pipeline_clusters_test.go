package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestMergePipelineClusters(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"foo": &resources.Pipeline{
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
					},
				},
			},
		},
	}

	err := bundle.Apply(context.Background(), b, mutator.MergePipelineClusters())
	assert.NoError(t, err)

	p := b.Config.Resources.Pipelines["foo"]

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
