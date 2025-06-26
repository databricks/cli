package tnresources

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestNewJobResource(t *testing.T) {
	client := &databricks.WorkspaceClient{}

	cfg := &resources.Job{
		JobSettings: jobs.JobSettings{
			Name: "test-job",
		},
	}

	res, cfgType, err := New(client, "jobs", "test-job", cfg)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure we received the correct resource type.
	require.IsType(t, &ResourceJob{}, res)
	require.IsType(t, reflect.TypeOf(ResourceJob{}.config), cfgType)
	require.IsType(t, reflect.TypeOf(jobs.JobSettings{}), cfgType)

	// The underlying config should match what we passed in.
	r := res.(*ResourceJob)
	require.Equal(t, cfg.JobSettings, r.config)
}
