package acceptance_test

import (
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const approvedFIPSModule = "v1.0.0"

// TestCLIBuiltWithFIPSModule asserts that the CLI was built against the validated
// cryptographic module. Without it, CI would still pass if GOFIPS140 stopped reaching
// the compiler: the suite would run against an ordinary build and report that FIPS works.
func TestCLIBuiltWithFIPSModule(t *testing.T) {
	// Required: the integration suite runs this package too (task integration passes
	// ./acceptance), and it runs from eng-dev-ecosystem, which does not set GOFIPS140.
	if os.Getenv("GOFIPS140") == "" {
		t.Skip("not a FIPS build")
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)

	execPath := BuildCLI(t, getBuildDir(t, cwd, runtime.GOOS, runtime.GOARCH), "", runtime.GOOS, runtime.GOARCH)

	out, err := exec.Command("go", "version", "-m", execPath).Output()
	require.NoError(t, err)
	buildInfo := string(out)

	// The stamped value carries a hash suffix (v1.0.0-<hash>) that moves with the
	// toolchain, so match the version prefix rather than the whole string.
	require.Contains(t, buildInfo, "GOFIPS140="+approvedFIPSModule,
		"binary was not built against the approved FIPS module")

	require.Contains(t, buildInfo, "DefaultGODEBUG=fips140=on",
		"FIPS module is linked but FIPS mode is not enabled by default")

	require.NotContains(t, buildInfo, "GOFIPS140=latest",
		"binary was built with an unvalidated module version")
}
