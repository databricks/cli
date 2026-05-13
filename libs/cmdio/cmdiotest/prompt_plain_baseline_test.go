package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_Plain pins prompts behavior when configured with only a
// Label (no Validate, no Mask). This is the most common shape used across
// cmd/auth and cmd/configure.
func TestPromptBaseline_Plain(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace name",
	})
	tm.WaitFor("Workspace name")
	tm.Golden("01-empty")

	tm.Type("hello")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Type("p there")
	tm.Golden("03-after-edit")

	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "help there", v, "snapshot:\n%s", tm.Snapshot())
}
