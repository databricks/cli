package aitools

import (
	"os"
	"testing"

	"github.com/databricks/cli/internal/testutil/hostenv"
)

// runTestHelper runs the helper process that a test re-executes the test
// binary as, and exits; it returns when the binary is not one. A test file sets
// it in init. It runs before Isolate, which would reset the environment that
// the test gave the helper.
var runTestHelper = func() {}

func TestMain(m *testing.M) {
	runTestHelper()
	finish := hostenv.Isolate()
	// Commands resolve the skills ref through the CLI compatibility manifest.
	// Use the embedded copy so tests that don't pin a ref skip the GitHub fetch
	// and its retries; tests of the fetch set this back to false.
	os.Setenv("DATABRICKS_FORCE_EMBEDDED_COMPAT", "true")
	os.Exit(finish(m.Run()))
}
