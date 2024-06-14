package jobs_test

import (
	"os"
	"testing"

	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobsSubmitInvalidJson(t *testing.T) {
	var jsonFlag flags.JsonFlag

	err := jsonFlag.Set("@testdata/job_cluster_test.json")
	require.NoError(t, err)

	var submitReq jobs.SubmitRun
	err = jsonFlag.Unmarshal(&submitReq)
	require.NoError(t, err)
}

func TestJobsSubmitInvalidJsonThroughYaml(t *testing.T) {
	path := "testdata/job_cluster_test.json"
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	v, err := yamlloader.LoadYAML(path, f)
	require.NoError(t, err)

	var submitReq jobs.SubmitRun
	nv, diag := convert.Normalize(&submitReq, v)

	assert.Empty(t, diag)
	assert.NotNil(t, nv)

}
