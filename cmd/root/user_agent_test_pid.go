package root

import (
	"context"
	"os"
	"strconv"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/useragent"
)

const (
	// TestPidEnvVar is the environment variable that enables PID injection into the user agent.
	// When set to "1", the CLI will include its process ID in the user agent string.
	// This is used by the test server to identify and signal the CLI process.
	TestPidEnvVar = "DATABRICKS_CLI_TEST_PID"
	testPidKey    = "test-pid"
)

// InjectTestPidToUserAgent adds the current process ID to the user agent if
// DATABRICKS_CLI_TEST_PID=1 is set. This enables the test server to identify
// and signal this process during acceptance tests.
func InjectTestPidToUserAgent(ctx context.Context) context.Context {
	if env.Get(ctx, TestPidEnvVar) != "1" {
		return ctx
	}
	return useragent.InContext(ctx, testPidKey, strconv.Itoa(os.Getpid()))
}
