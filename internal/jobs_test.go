package internal

import (
	"testing"
)

func TestCreateJob(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))
	RequireSuccessfulRun(t, "jobs", "create", "--json", "@testjsons/create_job_without_cluster.json", "--log-level=debug")
}
