package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecretBaseline_CtrlC pins Secret's behavior when the user cancels
// with Ctrl+C after typing a few characters.
func TestSecretBaseline_CtrlC(t *testing.T) {
	tm := termtest.NewSecret(t, "Personal access token")
	tm.WaitFor("Personal access token")
	tm.Golden("01-empty")

	tm.Type("abc")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyCtrlC)

	v, err := tm.Result()
	require.Error(t, err)
	t.Logf("error: %v", err)
	t.Logf("value: %q", v)
	assert.Empty(t, v, "snapshot:\n%s", tm.Snapshot())
}
