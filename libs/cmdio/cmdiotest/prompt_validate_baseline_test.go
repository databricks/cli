package cmdiotest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_Validate pins Prompt's behavior when a Validate
// callback is configured: validation re-runs on every keystroke, the
// indicator glyph reflects the result, and Enter is blocked while invalid.
func TestPromptBaseline_Validate(t *testing.T) {
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace host",
		Validate: func(s string) error {
			if !strings.Contains(s, "://") {
				return errors.New("must contain ://")
			}
			return nil
		},
	})
	tm.WaitFor("Workspace host")
	tm.Golden("01-empty")

	tm.Type("abc")
	tm.Golden("02-invalid-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Golden("03-cleared")

	tm.Type("https://example.com")
	tm.Golden("04-valid")

	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "https://example.com", v, "snapshot:\n%s", tm.Snapshot())
}
