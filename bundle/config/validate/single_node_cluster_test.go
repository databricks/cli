package validate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestValidateSingleNodeClusterFail(t *testing.T) {
	failCases := []struct {
		name string
		spec *compute.ClusterSpec
	}{
		{
			name: "no tags or conf",
			spec: &compute.ClusterSpec{
				ClusterName: "foo",
			},
		},
		{
			name: "no tags",
			spec: &compute.ClusterSpec{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "singleNode",
					"spark.master":                     "local[*]",
				},
			},
		},
		{
			name: "no conf",
			spec: &compute.ClusterSpec{
				CustomTags: map[string]string{
					"ResourceClass": "SingleNode",
				},
			},
		},
		{
			name: "invalid spark cluster profile",
			spec: &compute.ClusterSpec{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "invalid",
					"spark.master":                     "local[*]",
				},
				CustomTags: map[string]string{
					"ResourceClass": "SingleNode",
				},
			},
		},
		{
			name: "invalid spark.master",
			spec: &compute.ClusterSpec{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "singleNode",
					"spark.master":                     "invalid",
				},
				CustomTags: map[string]string{
					"ResourceClass": "SingleNode",
				},
			},
		},
		{
			name: "invalid tags",
			spec: &compute.ClusterSpec{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "singleNode",
					"spark.master":                     "local[*]",
				},
				CustomTags: map[string]string{
					"ResourceClass": "invalid",
				},
			},
		},
	}

	ctx := context.Background()

	// Test interactive clusters.
	for _, tc := range failCases {
		t.Run("interactive_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Clusters: map[string]*resources.Cluster{
							"foo": {
								ClusterSpec: tc.spec,
							},
						},
					},
				},
			}

			bundletest.SetLocation(b, "resources.clusters.foo", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Equal(t, diag.Diagnostics{
				{
					Severity:  diag.Warning,
					Summary:   singleNodeWarningSummary,
					Detail:    singleNodeWarningDetail,
					Locations: []dyn.Location{{File: "a.yml", Line: 1, Column: 1}},
					Paths:     []dyn.Path{dyn.NewPath(dyn.Key("resources"), dyn.Key("clusters"), dyn.Key("foo"))},
				},
			}, diags)
		})
	}

	// Test new job clusters.
	for _, tc := range failCases {
		t.Run("job_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"foo": {
								JobSettings: &jobs.JobSettings{
									JobClusters: []jobs.JobCluster{
										{
											NewCluster: *tc.spec,
										},
									},
								},
							},
						},
					},
				},
			}

			bundletest.SetLocation(b, "resources.jobs.foo.job_clusters[0]", []dyn.Location{{File: "b.yml", Line: 1, Column: 1}})

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Equal(t, diag.Diagnostics{
				{
					Severity:  diag.Warning,
					Summary:   singleNodeWarningSummary,
					Detail:    singleNodeWarningDetail,
					Locations: []dyn.Location{{File: "b.yml", Line: 1, Column: 1}},
					Paths:     []dyn.Path{dyn.MustPathFromString("resources.jobs.foo.job_clusters[0]")},
				},
			}, diags)

		})
	}

	// Test job task clusters.
	for _, tc := range failCases {
		t.Run("task_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"foo": {
								JobSettings: &jobs.JobSettings{
									Tasks: []jobs.Task{
										{
											NewCluster: tc.spec,
										},
									},
								},
							},
						},
					},
				},
			}

			bundletest.SetLocation(b, "resources.jobs.foo.tasks[0]", []dyn.Location{{File: "c.yml", Line: 1, Column: 1}})

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Equal(t, diag.Diagnostics{
				{
					Severity:  diag.Warning,
					Summary:   singleNodeWarningSummary,
					Detail:    singleNodeWarningDetail,
					Locations: []dyn.Location{{File: "c.yml", Line: 1, Column: 1}},
					Paths:     []dyn.Path{dyn.MustPathFromString("resources.jobs.foo.tasks[0].new_cluster")},
				},
			}, diags)
		})
	}
}

func TestValidateSingleNodeClusterPass(t *testing.T) {
	passCases := []struct {
		name string
		spec *compute.ClusterSpec
	}{
		{
			name: "single node cluster",
			spec: &compute.ClusterSpec{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "singleNode",
					"spark.master":                     "local[*]",
				},
				CustomTags: map[string]string{
					"ResourceClass": "SingleNode",
				},
			},
		},
		{
			name: "num workers is not zero",
			spec: &compute.ClusterSpec{
				NumWorkers: 1,
			},
		},
		{
			name: "autoscale is not nil",
			spec: &compute.ClusterSpec{
				Autoscale: &compute.AutoScale{
					MinWorkers: 1,
				},
			},
		},
		{
			name: "policy id is not empty",
			spec: &compute.ClusterSpec{
				PolicyId: "policy-abc",
			},
		},
	}

	ctx := context.Background()

	// Test interactive clusters.
	for _, tc := range passCases {
		t.Run("interactive_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Clusters: map[string]*resources.Cluster{
							"foo": {
								ClusterSpec: tc.spec,
							},
						},
					},
				},
			}

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Empty(t, diags)
		})
	}

	// Test new job clusters.
	for _, tc := range passCases {
		t.Run("job_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"foo": {
								JobSettings: &jobs.JobSettings{
									JobClusters: []jobs.JobCluster{
										{
											NewCluster: *tc.spec,
										},
									},
								},
							},
						},
					},
				},
			}

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Empty(t, diags)
		})
	}

	// Test job task clusters.
	for _, tc := range passCases {
		t.Run("task_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"foo": {
								JobSettings: &jobs.JobSettings{
									Tasks: []jobs.Task{
										{
											NewCluster: tc.spec,
										},
									},
								},
							},
						},
					},
				},
			}

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Empty(t, diags)
		})
	}
}

func TestValidateSingleNodePipelineClustersFail(t *testing.T) {
	failCases := []struct {
		name string
		spec pipelines.PipelineCluster
	}{
		{
			name: "no tags or conf",
			spec: pipelines.PipelineCluster{
				DriverInstancePoolId: "abcd",
			},
		},
		{
			name: "no tags",
			spec: pipelines.PipelineCluster{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "singleNode",
					"spark.master":                     "local[*]",
				},
			},
		},
		{
			name: "no conf",
			spec: pipelines.PipelineCluster{
				CustomTags: map[string]string{
					"ResourceClass": "SingleNode",
				},
			},
		},
		{
			name: "invalid spark cluster profile",
			spec: pipelines.PipelineCluster{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "invalid",
					"spark.master":                     "local[*]",
				},
				CustomTags: map[string]string{
					"ResourceClass": "SingleNode",
				},
			},
		},
		{
			name: "invalid spark.master",
			spec: pipelines.PipelineCluster{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "singleNode",
					"spark.master":                     "invalid",
				},
				CustomTags: map[string]string{
					"ResourceClass": "SingleNode",
				},
			},
		},
		{
			name: "invalid tags",
			spec: pipelines.PipelineCluster{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "singleNode",
					"spark.master":                     "local[*]",
				},
				CustomTags: map[string]string{
					"ResourceClass": "invalid",
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range failCases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Pipelines: map[string]*resources.Pipeline{
							"foo": {
								PipelineSpec: &pipelines.PipelineSpec{
									Clusters: []pipelines.PipelineCluster{
										tc.spec,
									},
								},
							},
						},
					},
				},
			}

			bundletest.SetLocation(b, "resources.pipelines.foo.clusters[0]", []dyn.Location{{File: "d.yml", Line: 1, Column: 1}})

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Equal(t, diag.Diagnostics{
				{
					Severity:  diag.Warning,
					Summary:   singleNodeWarningSummary,
					Detail:    singleNodeWarningDetail,
					Locations: []dyn.Location{{File: "d.yml", Line: 1, Column: 1}},
					Paths:     []dyn.Path{dyn.MustPathFromString("resources.pipelines.foo.clusters[0]")},
				},
			}, diags)
		})
	}
}

func TestValidateSingleNodePipelineClustersPass(t *testing.T) {
	passCases := []struct {
		name string
		spec pipelines.PipelineCluster
	}{
		{
			name: "single node cluster",
			spec: pipelines.PipelineCluster{
				SparkConf: map[string]string{
					"spark.databricks.cluster.profile": "singleNode",
					"spark.master":                     "local[*]",
				},
				CustomTags: map[string]string{
					"ResourceClass": "SingleNode",
				},
			},
		},
		{
			name: "num workers is not zero",
			spec: pipelines.PipelineCluster{
				NumWorkers: 1,
			},
		},
		{
			name: "autoscale is not nil",
			spec: pipelines.PipelineCluster{
				Autoscale: &pipelines.PipelineClusterAutoscale{
					MaxWorkers: 3,
				},
			},
		},
		{
			name: "policy id is not empty",
			spec: pipelines.PipelineCluster{
				PolicyId: "policy-abc",
			},
		},
	}

	ctx := context.Background()

	for _, tc := range passCases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Pipelines: map[string]*resources.Pipeline{
							"foo": {
								PipelineSpec: &pipelines.PipelineSpec{
									Clusters: []pipelines.PipelineCluster{
										tc.spec,
									},
								},
							},
						},
					},
				},
			}

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Empty(t, diags)
		})
	}
}
