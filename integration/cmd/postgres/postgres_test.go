// Package postgres_test contains integration tests for the experimental
// `databricks experimental postgres query` command. Skipped unless an
// autoscaling resource path or provisioned instance name is provided
// via DATABRICKS_POSTGRES_INTEGRATION_TARGET.
//
// To run locally against a real Lakebase endpoint:
//
//	export DATABRICKS_POSTGRES_INTEGRATION_TARGET=projects/foo/branches/main/endpoints/primary
//	go test ./integration/cmd/postgres/... -v
package postgres_test

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/databricks/cli/cmd/experimental"
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

	out := stdout.String()
	assert.Contains(t, out, `"x":1`)
}

func TestPostgresQuery_CommandOnly(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--output", "json", "SET search_path TO public")

	out := stdout.String()
	assert.Contains(t, out, `"command":"SET"`)
}

func TestPostgresQuery_TimeoutFires(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	// pg_sleep(5) with --timeout 1s should fail in well under 5s.
	start := time.Now()
	_, stderr, err := testcli.RequireErrorRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--timeout", "1s", "SELECT pg_sleep(5)")
	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second, "--timeout should cancel before pg_sleep finishes")
	assert.Contains(t, stderr.String(), "timed out after 1s")
}

func TestPostgresQuery_CancelOnInterrupt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Ctrl+C signal-driven cancel test is run via a separate harness on Windows")
	}
	requireTarget(t)
	t.Skip("manual: signal-driven cancel must be exercised with a child process; see plan section 'Cancellation and timeout'")
}

func TestPostgresQuery_StreamingCSV(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	// generate_series streams via pgx without buffering into memory; pick a
	// small-but-non-trivial bound so the test stays fast.
	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--output", "csv", "SELECT * FROM generate_series(1, 100) AS s")

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	assert.GreaterOrEqual(t, len(lines), 101, "expected header + 100 rows")
	assert.Equal(t, "s", lines[0])
}

func TestPostgresQuery_MultiInputJSON(t *testing.T) {
	target := requireTarget(t)
	ctx := t.Context()

	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "experimental", "postgres", "query",
		"--target", target, "--output", "json",
		"SELECT 1 AS a", "SELECT 2 AS b")

	out := stdout.String()
	assert.Contains(t, out, `"sql":"SELECT 1 AS a"`)
	assert.Contains(t, out, `"sql":"SELECT 2 AS b"`)
	assert.Contains(t, out, `"a":1`)
	assert.Contains(t, out, `"b":2`)
}
