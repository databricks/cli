package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func TestResolveComplexVariable(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
			},
			Variables: map[string]*variable.Variable{
				"cluster": {
					Value: map[string]any{
						"node_type_id": "Standard_DS3_v2",
						"num_workers":  2,
					},
					Type: variable.VariableTypeComplex,
				},
			},

			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									NewCluster: compute.ClusterSpec{
										NodeTypeId: "random",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()

	// Assign the variables to the dynamic configuration.
	diags := bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			var p dyn.Path
			var err error

			p = dyn.MustPathFromString("resources.jobs.job1.job_clusters[0]")
			v, err = dyn.SetByPath(v, p.Append(dyn.Key("new_cluster")), dyn.V("${var.cluster}"))
			require.NoError(t, err)

			return v, nil
		})
		return diag.FromErr(err)
	})
	require.NoError(t, diags.Error())

	diags = bundle.Apply(ctx, b, ResolveVariableReferences("bundle", "workspace", "variables"))
	require.NoError(t, diags.Error())
	require.Equal(t, "Standard_DS3_v2", b.Config.Resources.Jobs["job1"].JobSettings.JobClusters[0].NewCluster.NodeTypeId)
	require.Equal(t, 2, b.Config.Resources.Jobs["job1"].JobSettings.JobClusters[0].NewCluster.NumWorkers)
}

func TestResolveComplexVariableReferencesWithComplexVariablesError(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
			},
			Variables: map[string]*variable.Variable{
				"cluster": {
					Value: map[string]any{
						"node_type_id": "Standard_DS3_v2",
						"num_workers":  2,
						"spark_conf":   "${var.spark_conf}",
					},
					Type: variable.VariableTypeComplex,
				},
				"spark_conf": {
					Value: map[string]any{
						"spark.executor.memory": "4g",
						"spark.executor.cores":  "2",
					},
					Type: variable.VariableTypeComplex,
				},
			},

			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									NewCluster: compute.ClusterSpec{
										NodeTypeId: "random",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()

	// Assign the variables to the dynamic configuration.
	diags := bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			var p dyn.Path
			var err error

			p = dyn.MustPathFromString("resources.jobs.job1.job_clusters[0]")
			v, err = dyn.SetByPath(v, p.Append(dyn.Key("new_cluster")), dyn.V("${var.cluster}"))
			require.NoError(t, err)

			return v, nil
		})
		return diag.FromErr(err)
	})
	require.NoError(t, diags.Error())

	diags = bundle.Apply(ctx, b, bundle.Seq(ResolveVariableReferencesInComplexVariables(), ResolveVariableReferences("bundle", "workspace", "variables")))
	require.ErrorContains(t, diags.Error(), "complex variables cannot contain references to another complex variables")
}

func TestResolveComplexVariableWithVarReference(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
			},
			Variables: map[string]*variable.Variable{
				"package_version": {
					Value: "1.0.0",
				},
				"cluster_libraries": {
					Value: [](map[string]any){
						{
							"pypi": map[string]string{
								"package": "cicd_template==${var.package_version}",
							},
						},
					},
					Type: variable.VariableTypeComplex,
				},
			},

			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									Libraries: []compute.Library{},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()

	// Assign the variables to the dynamic configuration.
	diags := bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			var p dyn.Path
			var err error

			p = dyn.MustPathFromString("resources.jobs.job1.tasks[0]")
			v, err = dyn.SetByPath(v, p.Append(dyn.Key("libraries")), dyn.V("${var.cluster_libraries}"))
			require.NoError(t, err)

			return v, nil
		})
		return diag.FromErr(err)
	})
	require.NoError(t, diags.Error())

	diags = bundle.Apply(ctx, b, bundle.Seq(
		ResolveVariableReferencesInComplexVariables(),
		ResolveVariableReferences("bundle", "workspace", "variables"),
	))
	require.NoError(t, diags.Error())
	require.Equal(t, "cicd_template==1.0.0", b.Config.Resources.Jobs["job1"].JobSettings.Tasks[0].Libraries[0].Pypi.Package)
}

func TestResolveVariableReferencesWithSourceLinkedDeployment(t *testing.T) {
	testCases := []struct {
		enabled bool
		assert  func(t *testing.T, b *bundle.Bundle)
	}{
		{
			true,
			func(t *testing.T, b *bundle.Bundle) {
				// Variables that use workspace file path should have SyncRootValue during resolution phase
				require.Equal(t, "sync/root/path", b.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Configuration["source"])

				// The file path itself should remain the same
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
		{
			false,
			func(t *testing.T, b *bundle.Bundle) {
				require.Equal(t, "file/path", b.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Configuration["source"])
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
	}

	for _, testCase := range testCases {
		b := &bundle.Bundle{
			SyncRootPath: "sync/root/path",
			Config: config.Root{
				Presets: config.Presets{
					SourceLinkedDeployment: &testCase.enabled,
				},
				Workspace: config.Workspace{
					FilePath: "file/path",
				},
				Resources: config.Resources{
					Pipelines: map[string]*resources.Pipeline{
						"pipeline1": {
							PipelineSpec: &pipelines.PipelineSpec{
								Configuration: map[string]string{
									"source": "${workspace.file_path}",
								},
							},
						},
					},
				},
			},
		}

		diags := bundle.Apply(context.Background(), b, ResolveVariableReferences("workspace"))
		require.NoError(t, diags.Error())
		testCase.assert(t, b)
	}
}
