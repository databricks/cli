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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVariableReferences(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
			},
			Workspace: config.Workspace{
				RootPath: "${bundle.name}/bar",
				FilePath: "${workspace.root_path}/baz",
			},
		},
	}

	// Apply with an invalid prefix. This should not change the workspace root path.
	diags := bundle.Apply(context.Background(), b, ResolveVariableReferences("doesntexist"))
	require.Empty(t, diags)
	require.Equal(t, "${bundle.name}/bar", b.Config.Workspace.RootPath)
	require.Equal(t, "${workspace.root_path}/baz", b.Config.Workspace.FilePath)

	// Apply with a valid prefix. This should change the workspace root path.
	diags = bundle.Apply(context.Background(), b, ResolveVariableReferences("bundle", "workspace"))
	require.Empty(t, diags)
	require.Equal(t, "example/bar", b.Config.Workspace.RootPath)
	require.Equal(t, "example/bar/baz", b.Config.Workspace.FilePath)
}

func TestResolveVariableReferencesToBundleVariables(t *testing.T) {
	s := func(s string) *string {
		return &s
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
			},
			Workspace: config.Workspace{
				RootPath: "${bundle.name}/${var.foo}",
			},
			Variables: map[string]*variable.Variable{
				"foo": {
					Value: s("bar"),
				},
			},
		},
	}

	// Apply with a valid prefix. This should change the workspace root path.
	diags := bundle.Apply(context.Background(), b, ResolveVariableReferences("bundle", "variables"))
	require.Empty(t, diags)
	require.Equal(t, "example/bar", b.Config.Workspace.RootPath)
}

func TestResolveVariableReferencesToEmptyFields(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
				Git: config.Git{
					Branch: "",
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tags: map[string]string{
								"git_branch": "${bundle.git.branch}",
							},
						},
					},
				},
			},
		},
	}

	// Apply for the bundle prefix.
	diags := bundle.Apply(context.Background(), b, ResolveVariableReferences("bundle"))
	require.Empty(t, diags)

	// The job settings should have been interpolated to an empty string.
	require.Equal(t, "", b.Config.Resources.Jobs["job1"].JobSettings.Tags["git_branch"])
}

func TestResolveVariableReferencesForPrimitiveNonStringFields(t *testing.T) {
	var diags diag.Diagnostics

	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"no_alert_for_canceled_runs": {},
				"no_alert_for_skipped_runs":  {},
				"min_workers":                {},
				"max_workers":                {},
				"spot_bid_max_price":         {},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							NotificationSettings: &jobs.JobNotificationSettings{
								NoAlertForCanceledRuns: false,
								NoAlertForSkippedRuns:  false,
							},
							Tasks: []jobs.Task{
								{
									NewCluster: &compute.ClusterSpec{
										Autoscale: &compute.AutoScale{
											MinWorkers: 0,
											MaxWorkers: 0,
										},
										AzureAttributes: &compute.AzureAttributes{
											SpotBidMaxPrice: 0.0,
										},
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

	// Initialize the variables.
	diags = bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		err := b.Config.InitializeVariables([]string{
			"no_alert_for_canceled_runs=true",
			"no_alert_for_skipped_runs=true",
			"min_workers=1",
			"max_workers=2",
			"spot_bid_max_price=0.5",
		})
		return diag.FromErr(err)
	})
	require.NoError(t, diags.Error())

	// Assign the variables to the dynamic configuration.
	diags = bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			var p dyn.Path
			var err error

			// Set the notification settings.
			p = dyn.MustPathFromString("resources.jobs.job1.notification_settings")
			v, err = dyn.SetByPath(v, p.Append(dyn.Key("no_alert_for_canceled_runs")), dyn.V("${var.no_alert_for_canceled_runs}"))
			require.NoError(t, err)
			v, err = dyn.SetByPath(v, p.Append(dyn.Key("no_alert_for_skipped_runs")), dyn.V("${var.no_alert_for_skipped_runs}"))
			require.NoError(t, err)

			// Set the min and max workers.
			p = dyn.MustPathFromString("resources.jobs.job1.tasks[0].new_cluster.autoscale")
			v, err = dyn.SetByPath(v, p.Append(dyn.Key("min_workers")), dyn.V("${var.min_workers}"))
			require.NoError(t, err)
			v, err = dyn.SetByPath(v, p.Append(dyn.Key("max_workers")), dyn.V("${var.max_workers}"))
			require.NoError(t, err)

			// Set the spot bid max price.
			p = dyn.MustPathFromString("resources.jobs.job1.tasks[0].new_cluster.azure_attributes")
			v, err = dyn.SetByPath(v, p.Append(dyn.Key("spot_bid_max_price")), dyn.V("${var.spot_bid_max_price}"))
			require.NoError(t, err)

			return v, nil
		})
		return diag.FromErr(err)
	})
	require.NoError(t, diags.Error())

	// Apply for the variable prefix. This should resolve the variables to their values.
	diags = bundle.Apply(context.Background(), b, ResolveVariableReferences("variables"))
	require.NoError(t, diags.Error())
	assert.Equal(t, true, b.Config.Resources.Jobs["job1"].JobSettings.NotificationSettings.NoAlertForCanceledRuns)
	assert.Equal(t, true, b.Config.Resources.Jobs["job1"].JobSettings.NotificationSettings.NoAlertForSkippedRuns)
	assert.Equal(t, 1, b.Config.Resources.Jobs["job1"].JobSettings.Tasks[0].NewCluster.Autoscale.MinWorkers)
	assert.Equal(t, 2, b.Config.Resources.Jobs["job1"].JobSettings.Tasks[0].NewCluster.Autoscale.MaxWorkers)
	assert.Equal(t, 0.5, b.Config.Resources.Jobs["job1"].JobSettings.Tasks[0].NewCluster.AzureAttributes.SpotBidMaxPrice)
}
