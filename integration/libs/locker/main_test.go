package locker_test

import (
	"testing"

	"github.com/databricks/cli/integration/internal"
)

// TestMain is the entrypoint executed by the test runner.
// See [internal.WorkspaceMain] for prerequisites for running integration tests.
func TestMain(m *testing.M) {
	internal.WorkspaceMain(m)
}
