package ucm

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	ucmpkg "github.com/databricks/cli/ucm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_PolicyCheck_ValidFixturePasses(t *testing.T) {
	stdout, stderr, err := runVerb(t, validFixtureDir(t), "policy-check")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Policy check OK!")
}

func TestCmd_PolicyCheck_MissingTagFixtureFails(t *testing.T) {
	_, stderr, err := runVerb(t, filepath.Join("testdata", "missing_tag"), "policy-check")

	require.Error(t, err)
	assert.Contains(t, stderr, "requires tag")
}

// TestCmd_PolicyCheck_StrictOffAllowsWarnings verifies that a warning-only
// run is still considered a passing policy check when --strict is not set.
func TestCmd_PolicyCheck_StrictOffAllowsWarnings(t *testing.T) {
	withWarningSeed(t)

	stdout, _, err := runVerb(t, validFixtureDir(t), "policy-check")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Policy check OK!")
}

// TestCmd_PolicyCheck_StrictOnFailsOnWarning verifies that --strict promotes
// a warning diagnostic into a non-zero exit (matches `ucm validate --strict`).
func TestCmd_PolicyCheck_StrictOnFailsOnWarning(t *testing.T) {
	withWarningSeed(t)

	_, _, err := runVerb(t, validFixtureDir(t), "policy-check", "--strict")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Warnings are not allowed in strict mode")
}

// withWarningSeed installs a TestProcessHook so ProcessUcm emits exactly one
// diag.Warning into the runtime logdiag context for this test. Cleanup
// restores the previous hook (the package-level default is nil in production).
func withWarningSeed(t *testing.T) {
	t.Helper()
	prev := utils.TestProcessHook
	utils.TestProcessHook = func(ctx context.Context, u *ucmpkg.Ucm) {
		if prev != nil {
			prev(ctx, u)
		}
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "test-seeded warning",
		})
	}
	t.Cleanup(func() { utils.TestProcessHook = prev })
}
