package resourcemutator

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestInitializeNumWorkers(t *testing.T) {
	tests := []struct {
		name          string
		spec          compute.ClusterSpec
		wantForceSend bool
	}{
		{
			name:          "single-node cluster force-sends num_workers",
			spec:          compute.ClusterSpec{SparkVersion: "15.4.x-scala2.12", NodeTypeId: "i3.xlarge"},
			wantForceSend: true,
		},
		{
			name:          "autoscale cluster does not force-send",
			spec:          compute.ClusterSpec{Autoscale: &compute.AutoScale{MinWorkers: 1, MaxWorkers: 4}},
			wantForceSend: false,
		},
		{
			name:          "multi-node cluster does not force-send",
			spec:          compute.ClusterSpec{NumWorkers: 3},
			wantForceSend: false,
		},
		{
			name:          "already force-sent stays force-sent without duplicating",
			spec:          compute.ClusterSpec{ForceSendFields: []string{"NumWorkers"}},
			wantForceSend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := tt.spec
			initializeNumWorkers(&spec)

			count := 0
			for _, f := range spec.ForceSendFields {
				if f == "NumWorkers" {
					count++
				}
			}
			if tt.wantForceSend {
				assert.Equal(t, 1, count, "NumWorkers must appear in ForceSendFields exactly once")
			} else {
				assert.Equal(t, 0, count, "NumWorkers must not be in ForceSendFields")
			}
		})
	}
}

// TestPrepareJobSettingsForUpdateForcesNumWorkers locks the DECO-25361 fix: a
// single-node new_cluster must force-send num_workers on task-level clusters too,
// not just shared job_clusters. The terraform provider always sends num_workers:0
// for such clusters, so missing it on the task side made the direct engine
// produce a divergent create payload.
func TestPrepareJobSettingsForUpdateForcesNumWorkers(t *testing.T) {
	js := &jobs.JobSettings{
		Tasks: []jobs.Task{
			{
				TaskKey:    "single_node_task",
				NewCluster: &compute.ClusterSpec{SparkVersion: "15.4.x-scala2.12", NodeTypeId: "i3.xlarge"},
			},
			{
				TaskKey:    "autoscale_task",
				NewCluster: &compute.ClusterSpec{Autoscale: &compute.AutoScale{MinWorkers: 1, MaxWorkers: 4}},
			},
		},
		JobClusters: []jobs.JobCluster{
			{
				JobClusterKey: "single_node_cluster",
				NewCluster:    compute.ClusterSpec{SparkVersion: "15.4.x-scala2.12", NodeTypeId: "i3.xlarge"},
			},
		},
	}

	prepareJobSettingsForUpdate(js)

	assert.Contains(t, js.Tasks[0].NewCluster.ForceSendFields, "NumWorkers",
		"single-node task cluster must force-send num_workers")
	assert.NotContains(t, js.Tasks[1].NewCluster.ForceSendFields, "NumWorkers",
		"autoscale task cluster must not force-send num_workers")
	assert.Contains(t, js.JobClusters[0].NewCluster.ForceSendFields, "NumWorkers",
		"single-node job cluster must force-send num_workers")
}
