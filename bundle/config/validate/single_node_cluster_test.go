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
		name       string
		sparkConf  map[string]string
		customTags map[string]string
	}{
		{
			name: "no tags or conf",
		},
		{
			name: "no tags",
			sparkConf: map[string]string{
				"spark.databricks.cluster.profile": "singleNode",
				"spark.master":                     "local[*]",
			},
		},
		{
			name:       "no conf",
			customTags: map[string]string{"ResourceClass": "SingleNode"},
		},
		{
			name: "invalid spark cluster profile",
			sparkConf: map[string]string{
				"spark.databricks.cluster.profile": "invalid",
				"spark.master":                     "local[*]",
			},
			customTags: map[string]string{"ResourceClass": "SingleNode"},
		},
		{
			name: "invalid spark.master",
			sparkConf: map[string]string{
				"spark.databricks.cluster.profile": "singleNode",
				"spark.master":                     "invalid",
			},
			customTags: map[string]string{"ResourceClass": "SingleNode"},
		},
		{
			name: "invalid tags",
			sparkConf: map[string]string{
				"spark.databricks.cluster.profile": "singleNode",
				"spark.master":                     "local[*]",
			},
			customTags: map[string]string{"ResourceClass": "invalid"},
		},
		{
			name: "missing ResourceClass tag",
			sparkConf: map[string]string{
				"spark.databricks.cluster.profile": "singleNode",
				"spark.master":                     "local[*]",
			},
			customTags: map[string]string{"what": "ever"},
		},
		{
			name: "missing spark.master",
			sparkConf: map[string]string{
				"spark.databricks.cluster.profile": "singleNode",
			},
			customTags: map[string]string{"ResourceClass": "SingleNode"},
		},
		{
			name: "missing spark.databricks.cluster.profile",
			sparkConf: map[string]string{
				"spark.master": "local[*]",
			},
			customTags: map[string]string{"ResourceClass": "SingleNode"},
		},
	}

	ctx := context.Background()

	// Interactive clusters.
	for _, tc := range failCases {
		t.Run("interactive_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Clusters: map[string]*resources.Cluster{
							"foo": {
								ClusterSpec: &compute.ClusterSpec{
									SparkConf:  tc.sparkConf,
									CustomTags: tc.customTags,
								},
							},
						},
					},
				},
			}

			bundletest.SetLocation(b, "resources.clusters.foo", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

			// We can't set num_workers to 0 explicitly in the typed configuration.
			// Do it on the dyn.Value directly.
			bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
				return dyn.Set(v, "resources.clusters.foo.num_workers", dyn.V(0))
			})
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

	// Job clusters.
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
											NewCluster: compute.ClusterSpec{
												ClusterName: "my_cluster",
												SparkConf:   tc.sparkConf,
												CustomTags:  tc.customTags,
											},
										},
									},
								},
							},
						},
					},
				},
			}

			bundletest.SetLocation(b, "resources.jobs.foo.job_clusters[0].new_cluster", []dyn.Location{{File: "b.yml", Line: 1, Column: 1}})

			// We can't set num_workers to 0 explicitly in the typed configuration.
			// Do it on the dyn.Value directly.
			bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
				return dyn.Set(v, "resources.jobs.foo.job_clusters[0].new_cluster.num_workers", dyn.V(0))
			})

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Equal(t, diag.Diagnostics{
				{
					Severity:  diag.Warning,
					Summary:   singleNodeWarningSummary,
					Detail:    singleNodeWarningDetail,
					Locations: []dyn.Location{{File: "b.yml", Line: 1, Column: 1}},
					Paths:     []dyn.Path{dyn.MustPathFromString("resources.jobs.foo.job_clusters[0].new_cluster")},
				},
			}, diags)

		})
	}

	// Job task clusters.
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
											NewCluster: &compute.ClusterSpec{
												ClusterName: "my_cluster",
												SparkConf:   tc.sparkConf,
												CustomTags:  tc.customTags,
											},
										},
									},
								},
							},
						},
					},
				},
			}

			bundletest.SetLocation(b, "resources.jobs.foo.tasks[0].new_cluster", []dyn.Location{{File: "c.yml", Line: 1, Column: 1}})

			// We can't set num_workers to 0 explicitly in the typed configuration.
			// Do it on the dyn.Value directly.
			bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
				return dyn.Set(v, "resources.jobs.foo.tasks[0].new_cluster.num_workers", dyn.V(0))
			})

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

	// Pipeline clusters.
	for _, tc := range failCases {
		t.Run("pipeline_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Pipelines: map[string]*resources.Pipeline{
							"foo": {
								PipelineSpec: &pipelines.PipelineSpec{
									Clusters: []pipelines.PipelineCluster{
										{
											SparkConf:  tc.sparkConf,
											CustomTags: tc.customTags,
										},
									},
								},
							},
						},
					},
				},
			}

			bundletest.SetLocation(b, "resources.pipelines.foo.clusters[0]", []dyn.Location{{File: "d.yml", Line: 1, Column: 1}})

			// We can't set num_workers to 0 explicitly in the typed configuration.
			// Do it on the dyn.Value directly.
			bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
				return dyn.Set(v, "resources.pipelines.foo.clusters[0].num_workers", dyn.V(0))
			})

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

func TestValidateSingleNodeClusterPass(t *testing.T) {
	zero := 0
	one := 1

	passCases := []struct {
		name       string
		numWorkers *int
		sparkConf  map[string]string
		customTags map[string]string
		policyId   string
	}{
		{
			name: "single node cluster",
			sparkConf: map[string]string{
				"spark.databricks.cluster.profile": "singleNode",
				"spark.master":                     "local[*]",
			},
			customTags: map[string]string{
				"ResourceClass": "SingleNode",
			},
			numWorkers: &zero,
		},
		{
			name:       "num workers is not zero",
			numWorkers: &one,
		},
		{
			name: "num workers is not set",
		},
		{
			name:       "policy id is not empty",
			policyId:   "policy-abc",
			numWorkers: &zero,
		},
	}

	ctx := context.Background()

	// Interactive clusters.
	for _, tc := range passCases {
		t.Run("interactive_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Clusters: map[string]*resources.Cluster{
							"foo": {
								ClusterSpec: &compute.ClusterSpec{
									SparkConf:  tc.sparkConf,
									CustomTags: tc.customTags,
									PolicyId:   tc.policyId,
								},
							},
						},
					},
				},
			}

			if tc.numWorkers != nil {
				bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
					return dyn.Set(v, "resources.clusters.foo.num_workers", dyn.V(*tc.numWorkers))
				})
			}

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Empty(t, diags)
		})
	}

	// Job clusters.
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
											NewCluster: compute.ClusterSpec{
												ClusterName: "my_cluster",
												SparkConf:   tc.sparkConf,
												CustomTags:  tc.customTags,
												PolicyId:    tc.policyId,
											},
										},
									},
								},
							},
						},
					},
				},
			}

			if tc.numWorkers != nil {
				bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
					return dyn.Set(v, "resources.jobs.foo.job_clusters[0].new_cluster.num_workers", dyn.V(*tc.numWorkers))
				})
			}

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Empty(t, diags)
		})
	}

	// Job task clusters.
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
											NewCluster: &compute.ClusterSpec{
												ClusterName: "my_cluster",
												SparkConf:   tc.sparkConf,
												CustomTags:  tc.customTags,
												PolicyId:    tc.policyId,
											},
										},
									},
								},
							},
						},
					},
				},
			}

			if tc.numWorkers != nil {
				bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
					return dyn.Set(v, "resources.jobs.foo.tasks[0].new_cluster.num_workers", dyn.V(*tc.numWorkers))
				})
			}

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Empty(t, diags)
		})
	}

	// Pipeline clusters.
	for _, tc := range passCases {
		t.Run("pipeline_"+tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Pipelines: map[string]*resources.Pipeline{
							"foo": {
								PipelineSpec: &pipelines.PipelineSpec{
									Clusters: []pipelines.PipelineCluster{
										{
											SparkConf:  tc.sparkConf,
											CustomTags: tc.customTags,
											PolicyId:   tc.policyId,
										},
									},
								},
							},
						},
					},
				},
			}

			if tc.numWorkers != nil {
				bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
					return dyn.Set(v, "resources.pipelines.foo.clusters[0].num_workers", dyn.V(*tc.numWorkers))
				})
			}

			diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), SingleNodeCluster())
			assert.Empty(t, diags)
		})
	}
}
