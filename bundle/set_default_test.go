package bundle

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultSkipsPureReference(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  jobs:
    base:
      schedule:
        quartz_cron_expression: "0 0 * * * ?"
    viaref:
      schedule: ${resources.jobs.base.schedule}
`))
	require.Empty(t, diags)
	b := &Bundle{Config: *root}

	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	SetDefault(ctx, b, "resources.jobs.*.schedule.pause_status", "UNPAUSED")
	require.Empty(t, logdiag.FlushCollected(ctx))

	v := b.Config.Value()
	base, err := dyn.GetByPath(v, dyn.MustPathFromString("resources.jobs.base.schedule.pause_status"))
	require.NoError(t, err)
	assert.Equal(t, "UNPAUSED", base.MustString())

	viaref, err := dyn.GetByPath(v, dyn.MustPathFromString("resources.jobs.viaref.schedule"))
	require.NoError(t, err)
	assert.Equal(t, "${resources.jobs.base.schedule}", viaref.MustString())
}

func TestSetDefaultSkipsPureReferenceAboveKey(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		pattern string
		set     string
		ref     string
	}{
		{
			name: "job_cluster_new_cluster",
			config: `
resources:
  jobs:
    base:
      job_clusters:
        - job_cluster_key: main
          new_cluster:
            workload_type:
              clients:
                jobs: false
    viaref:
      job_clusters:
        - job_cluster_key: main
          new_cluster: ${resources.jobs.base.job_clusters[0].new_cluster}
`,
			pattern: "resources.jobs.*.job_clusters[*].new_cluster.workload_type.clients.notebooks",
			set:     "resources.jobs.base.job_clusters[0].new_cluster.workload_type.clients.notebooks",
			ref:     "resources.jobs.viaref.job_clusters[0].new_cluster",
		},
		{
			name: "cluster_workload_type",
			config: `
resources:
  clusters:
    base:
      workload_type:
        clients:
          jobs: false
    viaref:
      workload_type: ${resources.clusters.base.workload_type}
`,
			pattern: "resources.clusters.*.workload_type.clients.notebooks",
			set:     "resources.clusters.base.workload_type.clients.notebooks",
			ref:     "resources.clusters.viaref.workload_type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(tc.config))
			require.Empty(t, diags)
			b := &Bundle{Config: *root}

			ctx := logdiag.InitContext(t.Context())
			logdiag.SetCollect(ctx, true)
			SetDefault(ctx, b, tc.pattern, true)
			require.Empty(t, logdiag.FlushCollected(ctx))

			v := b.Config.Value()
			set, err := dyn.GetByPath(v, dyn.MustPathFromString(tc.set))
			require.NoError(t, err)
			assert.True(t, set.MustBool())

			ref, err := dyn.GetByPath(v, dyn.MustPathFromString(tc.ref))
			require.NoError(t, err)
			assert.True(t, dynvar.IsPureVariableReference(ref.MustString()))
		})
	}
}

func TestSetDefaultErrorHasFullPath(t *testing.T) {
	tests := []struct {
		name          string
		workloadType  dyn.Value
		expectedError string
	}{
		{
			name:          "mismatch_inside_pattern",
			workloadType:  dyn.V("not a map"),
			expectedError: `resources.jobs.job.job_clusters[0].new_cluster.workload_type: expected a map to index "clients", found string`,
		},
		{
			name:          "mismatch_at_pattern_end",
			workloadType:  dyn.V(map[string]dyn.Value{"clients": dyn.V("x")}),
			expectedError: `resources.jobs.job.job_clusters[0].new_cluster.workload_type.clients: expected a map to index "notebooks", found string`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Built directly because loading the config would drop a wrong-kind workload_type.
			v := dyn.V(map[string]dyn.Value{
				"resources": dyn.V(map[string]dyn.Value{
					"jobs": dyn.V(map[string]dyn.Value{
						"job": dyn.V(map[string]dyn.Value{
							"job_clusters": dyn.V([]dyn.Value{
								dyn.V(map[string]dyn.Value{
									"new_cluster": dyn.V(map[string]dyn.Value{
										"workload_type": tc.workloadType,
									}),
								}),
							}),
						}),
					}),
				}),
			})

			pattern := dyn.MustPatternFromString("resources.jobs.*.job_clusters[*].new_cluster.workload_type.clients")
			m := SetDefaultMutator(pattern, "notebooks", true).(*setDefault)
			_, err := m.apply(v, dyn.EmptyPath, m.pattern)
			assert.EqualError(t, err, tc.expectedError)
		})
	}
}
