package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecretBaseline_Typing pins secrets behavior:
// each typed character should render as the configured mask ('*'), backspace
// should erase one mask char, and Enter should return the typed value.
func TestSecretBaseline_Typing(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSecret(t, "Enter password")
	tm.WaitFor("Enter password")
	tm.Golden("01-empty")

	tm.Type("hunter2")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Golden("03-after-backspace")

	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hunter", v, "snapshot:\n%s", tm.Snapshot())
}
