package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_Mask pins runprompts behavior
// when configured with Mask='*'. This is the shape used by `databricks
// configure` for personal access token entry (cmd/configure/configure.go:46).
func TestPromptBaseline_Mask(t *testing.T) {
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Personal access token",
		Mask:  '*',
	})
	tm.WaitFor("Personal access token")
	tm.Golden("01-empty")

	tm.Type("dapi-secret")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Golden("03-after-backspace")

	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "dapi-sec", v, "snapshot:\n%s", tm.Snapshot())
}
