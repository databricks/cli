package storage_credentials_integration

import (
	"testing"

	"github.com/databricks/cli/integration/internal"
)

// TestMain is the entrypoint executed by the test runner.
// See [internal.Main] for prerequisites for running integration tests.
func TestMain(m *testing.M) {
	internal.Main(m)
}
