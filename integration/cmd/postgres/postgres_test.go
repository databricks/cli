// Package postgres_test contains integration tests for the experimental
// `databricks experimental postgres query` command. Skipped unless an
// autoscaling resource path or provisioned instance name is provided
// via DATABRICKS_POSTGRES_INTEGRATION_TARGET.
//
// To run locally against a real Lakebase endpoint, set both the standard
// auth env (DATABRICKS_HOST + DATABRICKS_TOKEN, or a configured profile)
// and the target:
//
//	export DATABRICKS_POSTGRES_INTEGRATION_TARGET=projects/foo/branches/main/endpoints/primary
//	go test ./integration/cmd/postgres/... -v
//
// Ctrl+C cancellation is intentionally not in this suite: it requires a
// child-process harness (the test runner cannot share signal handlers
// with the in-process command). Tracked as a follow-up.
package postgres_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/internal/testcli"
)

// targetEnv is the env var that gates these tests. Either a provisioned
// instance name or an autoscaling resource path; the command picks the
// right resolver based on the leading "projects/" segment.
const targetEnv = "DATABRICKS_POSTGRES_INTEGRATION_TARGET"

func requireTarget(t *testing.T) string {
	target := os.Getenv(targetEnv)
	if target == "" {
		t.Skipf("set %s to run postgres integration tests", targetEnv)
	}
	return target
}

func TestPostgresQuery_SimpleSelect(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--output", "json", "SELECT 1 AS x")

	// Parsing the JSON instead of substring-matching makes the test robust
	// to encoder formatting drift (whitespace, key order).
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &rows))
	require.Len(t, rows, 1)
	assert.EqualValues(t, 1, rows[0]["x"])
}

func TestPostgresQuery_CommandOnly(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--output", "json", "SET search_path TO public")

	var obj map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &obj))
	assert.Equal(t, "SET", obj["command"])
}

func TestPostgresQuery_TimeoutFires(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	// Warm up first: pay the auth + connect (and potential cold-start)
	// cost before timing the --timeout assertion. Without this, a cold
	// Lakebase autoscaling endpoint could push the timed run past any
	// reasonable deadline even though --timeout did exactly the right
	// thing. Now `start` measures what we care about: how long the
	// 1-second deadline takes to actually cancel the in-flight statement.
	testcli.RequireSuccessfulRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--output", "json", "SELECT 1")

	// pg_sleep(5) with --timeout 1s should fail well within the watcher's
	// 5s DeadlineDelay. <5s rules out a silent regression to the
	// TCP-keepalive timeout (~minutes).
	start := time.Now()
	_, stderr, _ := testcli.RequireErrorRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--timeout", "1s", "SELECT pg_sleep(5)")
	assert.Less(t, time.Since(start), 5*time.Second, "--timeout should cancel before pg_sleep finishes")
	assert.Contains(t, stderr.String(), "timed out after 1s")
}

func TestPostgresQuery_StreamingCSV(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	// 100k rows is large enough to exercise streaming under realistic memory
	// pressure (the buffered text path would still complete but allocate
	// the whole result; the streaming CSV path keeps allocations bounded).
	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--output", "csv", "SELECT * FROM generate_series(1, 100000) AS s")

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	assert.GreaterOrEqual(t, len(lines), 100001, "expected header + 100000 rows")
	assert.Equal(t, "s", lines[0])
}

func TestPostgresQuery_MultiInputJSON(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--output", "json",
		"SELECT 1 AS a", "SELECT 2 AS b")

	var results []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &results))
	require.Len(t, results, 2)
	assert.Equal(t, "SELECT 1 AS a", results[0]["sql"])
	assert.Equal(t, "SELECT 2 AS b", results[1]["sql"])
}
