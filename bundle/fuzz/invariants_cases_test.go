package fuzz

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

// recordingT captures whether the invariant assertions failed, so the table below
// can check that a bad payload is rejected and a good one is accepted without a
// real deploy.
type recordingT struct{ failed bool }

func (r *recordingT) Errorf(string, ...any) { r.failed = true }

// FailNow is only reached if decodePayload errors; every case here is valid JSON,
// so record and stop the goroutine the way require would.
func (r *recordingT) FailNow() { panic("unexpected FailNow") }

func TestCheckJobInvariants(t *testing.T) {
	job := &resources.Job{
		JobSettings: jobs.JobSettings{
			Name: "j",
			JobClusters: []jobs.JobCluster{
				{JobClusterKey: "shared", NewCluster: compute.ClusterSpec{}},
			},
			Tasks: []jobs.Task{
				{TaskKey: "a"},
				{TaskKey: "b"},
			},
		},
	}

	tests := []struct {
		name       string
		payload    string
		wantFailed bool
	}{
		{
			name:    "valid payload",
			payload: `{"name":"j","job_clusters":[{"job_cluster_key":"shared","new_cluster":{"num_workers":0}}],"tasks":[{"task_key":"a","job_cluster_key":"shared"},{"task_key":"b","depends_on":[{"task_key":"a"}]}]}`,
		},
		{
			name:       "renamed job",
			payload:    `{"name":"other","job_clusters":[{"job_cluster_key":"shared"}],"tasks":[{"task_key":"a"},{"task_key":"b"}]}`,
			wantFailed: true,
		},
		{
			name:       "dropped task",
			payload:    `{"name":"j","job_clusters":[{"job_cluster_key":"shared"}],"tasks":[{"task_key":"a"}]}`,
			wantFailed: true,
		},
		{
			name:       "dangling dependency",
			payload:    `{"name":"j","job_clusters":[{"job_cluster_key":"shared"}],"tasks":[{"task_key":"a"},{"task_key":"b","depends_on":[{"task_key":"ghost"}]}]}`,
			wantFailed: true,
		},
		{
			name:       "dangling job cluster reference",
			payload:    `{"name":"j","job_clusters":[{"job_cluster_key":"shared"}],"tasks":[{"task_key":"a","job_cluster_key":"missing"},{"task_key":"b"}]}`,
			wantFailed: true,
		},
		{
			name:    "new_cluster without explicit size is a valid single node",
			payload: `{"name":"j","job_clusters":[{"job_cluster_key":"shared","new_cluster":{"spark_version":"x"}}],"tasks":[{"task_key":"a"},{"task_key":"b"}]}`,
		},
		{
			name:    "single-node new_cluster with num_workers 0",
			payload: `{"name":"j","job_clusters":[{"job_cluster_key":"shared","new_cluster":{"num_workers":0}}],"tasks":[{"task_key":"a"},{"task_key":"b"}]}`,
		},
		{
			name:    "autoscale new_cluster",
			payload: `{"name":"j","job_clusters":[{"job_cluster_key":"shared","new_cluster":{"autoscale":{"min_workers":1,"max_workers":3}}}],"tasks":[{"task_key":"a"},{"task_key":"b"}]}`,
		},
		{
			name:       "new_cluster sets both autoscale and num_workers",
			payload:    `{"name":"j","job_clusters":[{"job_cluster_key":"shared","new_cluster":{"autoscale":{"min_workers":1,"max_workers":3},"num_workers":2}}],"tasks":[{"task_key":"a"},{"task_key":"b"}]}`,
			wantFailed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recordingT{}
			checkJobInvariants(rec, 0, job, json.RawMessage(tt.payload))
			assert.Equal(t, tt.wantFailed, rec.failed)
		})
	}
}
