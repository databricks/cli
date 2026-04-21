package ucm

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/dyn/jsonsaver"
	"github.com/databricks/cli/libs/logdiag"
	ucmpkg "github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/deploy/terraform/tfdyn"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateSmokeGolden toggles rewrite of the plan-smoke golden file when a
// deliberate change to the converter output warrants it. Pass -update to the
// test binary to regenerate. Mirrors libs/testdiff's OverwriteMode pattern
// but scoped to this test file so the global flag surface stays unchanged.
var updateSmokeGolden = flag.Bool("update-smoke", false, "regenerate cmd/ucm/testdata/deploy_smoke/expected.tf.json.golden")

// TestCmd_PlanSmoke_EndToEnd exercises the full ucm.yml → phases → tf JSON
// pipeline against cmd/ucm/testdata/deploy_smoke. It is the M1 close-out
// fixture: minimal project, nested form with tag inheritance, one grant.
//
// The test does NOT stand up a real terraform binary. It invokes the load and
// mutator chain via phases.LoadDefaultTarget, then drives tfdyn.Convert
// directly and diffs the marshalled JSON against a committed golden file.
//
// Companion to TestCmd_Plan_HappyPathPrintsSummary (verb wiring coverage)
// which uses the fake-tf harness to prove the cobra path end-to-end; this
// test guarantees the converter side of the same pipeline stays stable.
func TestCmd_PlanSmoke_EndToEnd(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	fixture := filepath.Join("testdata", "deploy_smoke")

	u, err := ucmpkg.Load(ctx, fixture)
	require.NoError(t, err)
	require.NotNil(t, u)

	phases.LoadDefaultTarget(ctx, u)
	require.False(t, logdiag.HasError(ctx), "LoadDefaultTarget reported errors")
	require.NotNil(t, u.Target, "expected default target to be selected")

	assertSmokeGolden(t, ctx, u)
}

// assertSmokeGolden runs the converter, marshals the tree the same way
// Render does in production, and asserts against the committed golden file.
// When -update-smoke is passed the test overwrites the golden file instead
// of failing — keep the toggle close to the assertion to make regeneration
// obvious in diff review.
func assertSmokeGolden(t *testing.T, ctx context.Context, u *ucmpkg.Ucm) {
	t.Helper()

	tree, err := tfdyn.Convert(ctx, u)
	require.NoError(t, err)

	got, err := jsonsaver.MarshalIndent(tree, "", "  ")
	require.NoError(t, err)

	goldenPath := filepath.Join("testdata", "deploy_smoke", "expected.tf.json.golden")
	if *updateSmokeGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o644))
		t.Logf("wrote %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "read golden (pass -update-smoke to regenerate)")
	assert.JSONEq(t, string(want), string(got), "rendered tf JSON diverged from golden; run with -update-smoke to regenerate")
}

// TestCmd_PlanSmoke_VerbHappyPath drives the cobra plan verb against the same
// smoke fixture with the fake-tf harness. Complements TestCmd_PlanSmoke_EndToEnd
// by proving the full CLI pivot (ucm plan → phases.Plan → TerraformWrapper)
// stays wired once PreRunE auth is stripped out for tests.
func TestCmd_PlanSmoke_VerbHappyPath(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.PlanResult = &terraform.PlanResult{HasChanges: true, Summary: "smoke plan ready"}

	stdout, stderr, err := runVerb(t, filepath.Join("testdata", "deploy_smoke"), "plan")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "smoke plan ready")
	assert.Equal(t, 1, h.tf.RenderCalls)
	assert.Equal(t, 1, h.tf.InitCalls)
	assert.Equal(t, 1, h.tf.PlanCalls)
}
