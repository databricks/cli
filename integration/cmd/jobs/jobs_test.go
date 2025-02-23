package jobs_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateJob(t *testing.T) {
	testutil.Require(t, testutil.Azure)
	ctx := context.Background()
	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "jobs", "create", "--json", "@testdata/create_job_without_workers.json", "--log-level=debug")
	assert.Empty(t, stderr.String())
	var output map[string]int
	err := json.Unmarshal(stdout.Bytes(), &output)
	require.NoError(t, err)
	testcli.RequireSuccessfulRun(t, ctx, "jobs", "delete", strconv.Itoa(output["job_id"]), "--log-level=debug")
}
