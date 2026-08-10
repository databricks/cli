package acceptance_test

import (
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// approvedFIPSModule is the frozen Go Cryptographic Module version that has
// completed CMVP validation. GOFIPS140=latest also enables FIPS mode, but tracks
// the in-tree source and moves with every Go release, so it leaves no fixed
// artifact to cite; only a frozen version is accepted here.
const approvedFIPSModule = "v1.0.0"

// TestCLIBuiltWithFIPSModule asserts that the binary the acceptance suite just
// exercised really was built against the validated module.
//
// Without this, the FIPS CI job would still pass if GOFIPS140 silently stopped
// reaching the compiler -- the suite would run happily against an ordinary
// build and report that FIPS "works". The setting is only observable in the
// binary's own build info, so read it back from there.
//
// Skipped unless GOFIPS140 is set, so the default build is unaffected.
func TestCLIBuiltWithFIPSModule(t *testing.T) {
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

	// GOFIPS140 links the module and defaults FIPS mode on. Both matter: a binary
	// that links the module but leaves the mode off behaves like a plain build.
	require.Contains(t, buildInfo, "DefaultGODEBUG=fips140=on",
		"FIPS module is linked but FIPS mode is not enabled by default")

	require.NotContains(t, buildInfo, "GOFIPS140=latest",
		"binary was built with an unvalidated module version")
}
