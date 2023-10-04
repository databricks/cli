package internal

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/stretchr/testify/assert"
)

func TestAccCreateJob(t *testing.T) {
	acc.WorkspaceTest(t)
	env := GetEnvOrSkipTest(t, "CLOUD_ENV")
	if env != "azure" {
		t.Skipf("Not running test on cloud %s", env)
	}
	stdout, stderr := RequireSuccessfulRun(t, "jobs", "create", "--json", "@testjsons/create_job_without_cluster.json", "--log-level=debug")
	assert.Equal(t, "", stderr.String())
	var output map[string]int
	err := json.Unmarshal(stdout.Bytes(), &output)
	assert.NoError(t, err)
	RequireSuccessfulRun(t, "jobs", "delete", fmt.Sprint(output["job_id"]), "--log-level=debug")
}
