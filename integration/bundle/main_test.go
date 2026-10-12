package bundle_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/databricks/cli/internal/testutil"
)

// TestMain stages a local Terraform filesystem mirror via
// acceptance/install_terraform.py before any test in this package runs.
//
// The Databricks-internal CI runners that execute these tests cannot reach
// registry.terraform.io, so without this setup `bundle deploy` fails inside
// `terraform init` with "could not connect to registry.terraform.io: ... EOF".
// Routing through the local mirror also makes a plain `go test ./integration/bundle/...`
// work outside CI without any extra environment variables.
func TestMain(m *testing.M) {
	if err := testutil.SetupTerraform(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set up Terraform for integration tests: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
