// Only build this package if the dbr tag is set.
// We do not want to run this test on normal integration test runs.
//go:build dbr

package dbr

import "testing"

func TestAcceptDbr(t *testing.T) {
	// Build the CLI archives
	RunCommand(t, []string{"go", "run", ".", "archive", "archive/cli.tar.gz"}, ".")
}
