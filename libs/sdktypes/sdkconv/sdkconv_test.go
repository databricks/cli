package sdkconv_test

import (
	"encoding/json"
	"testing"

	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/sdkconv"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jobJSON = `{"job_id":123,"settings":{"name":"x","max_concurrent_runs":0,"tasks":[{"task_key":"a","notebook_task":{"notebook_path":"/p"},"existing_cluster_id":"c1","environment_key":"e"}],"run_as":{"service_principal_name":"sp"},"format":"MULTI_TASK"}}`

func TestRoundTripMatchesGenerated(t *testing.T) {
	var m jobs.GetJobResponse
	require.NoError(t, json.Unmarshal([]byte(jobJSON), &m))

	// ToSDK exists for Job (it is decoded from responses); compare with sdkconv.
	want, err := jobs.GetJobResponseToSDK(&m)
	require.NoError(t, err)
	got, err := sdkconv.ToSDK[sdkjobs.GetJobResponse](&m)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	// FromSDK does not exist for Job; sdkconv provides it.
	back, err := sdkconv.FromSDK[jobs.GetJobResponse](got)
	require.NoError(t, err)
	out, err := json.Marshal(back)
	require.NoError(t, err)
	assert.JSONEq(t, jobJSON, string(out))
}

func TestMultipleOneofMembers(t *testing.T) {
	m := jobs.JobRunAs{UserName: new("u"), ServicePrincipalName: new("sp")}
	_, err := sdkconv.ToSDK[sdkjobs.JobRunAs](&m)
	assert.ErrorContains(t, err, "multiple oneof members set")
}

func TestNil(t *testing.T) {
	got, err := sdkconv.FromSDK[jobs.GetJobResponse]((*sdkjobs.GetJobResponse)(nil))
	require.NoError(t, err)
	assert.Nil(t, got)
}
