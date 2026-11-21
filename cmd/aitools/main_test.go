package aitools

import (
	"os"
	"testing"

	"github.com/databricks/cli/internal/testutil/hostenv"
)

func TestMain(m *testing.M) {
	finish := hostenv.Isolate()
	// Commands resolve the skills ref through the CLI compatibility manifest.
	// Use the embedded copy so tests that don't pin a ref skip the GitHub fetch
	// and its retries; tests of the fetch set this back to false.
	os.Setenv("DATABRICKS_FORCE_EMBEDDED_COMPAT", "true")
	os.Exit(finish(m.Run()))
}
