package resourcemutator_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/stretchr/testify/assert"
)

func TestMergePipelineClusters(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"foo": {
						CreatePipelineRequest: pipelines.CreatePipelineRequest{
							Clusters: []pipelines.PipelineCluster{
								{
									NodeTypeId: new("i3.xlarge"),
									NumWorkers: new(2),
									PolicyId:   new("1234"),
								},
								{
									Label:      new("maintenance"),
									NodeTypeId: new("i3.2xlarge"),
								},
								{
									NodeTypeId: new("i3.2xlarge"),
									NumWorkers: new(4),
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergePipelineClusters())
	assert.NoError(t, diags.Error())

	p := b.Config.Resources.Pipelines["foo"]

	assert.Len(t, p.Clusters, 2)
	assert.Equal(t, "default", utils.Deref(p.Clusters[0].Label))
	assert.Equal(t, "maintenance", utils.Deref(p.Clusters[1].Label))

	// The default cluster was merged with a subsequent one.
	pc0 := p.Clusters[0]
	assert.Equal(t, "i3.2xlarge", utils.Deref(pc0.NodeTypeId))
	assert.Equal(t, 4, utils.Deref(pc0.NumWorkers))
	assert.Equal(t, "1234", utils.Deref(pc0.PolicyId))

	// The maintenance cluster was left untouched.
	pc1 := p.Clusters[1]
	assert.Equal(t, "i3.2xlarge", utils.Deref(pc1.NodeTypeId))
}

func TestMergePipelineClustersCaseInsensitive(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"foo": {
						CreatePipelineRequest: pipelines.CreatePipelineRequest{
							Clusters: []pipelines.PipelineCluster{
								{
									Label:      new("default"),
									NumWorkers: new(2),
								},
								{
									Label:      new("DEFAULT"),
									NumWorkers: new(4),
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergePipelineClusters())
	assert.NoError(t, diags.Error())

	p := b.Config.Resources.Pipelines["foo"]
	assert.Len(t, p.Clusters, 1)

	// The default cluster was merged with a subsequent one.
	pc0 := p.Clusters[0]
	assert.Equal(t, "default", strings.ToLower(utils.Deref(pc0.Label)))
	assert.Equal(t, 4, utils.Deref(pc0.NumWorkers))
}

func TestMergePipelineClustersNilPipelines(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: nil,
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergePipelineClusters())
	assert.NoError(t, diags.Error())
}

func TestMergePipelineClustersEmptyPipelines(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergePipelineClusters())
	assert.NoError(t, diags.Error())
}
