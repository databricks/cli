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
	t.Parallel()
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

// TestPromptBaseline_ValidateEnterBeforeEdit pins that Enter on an untouched
// prompt flips the glyph to "✗" along with the error line.
func TestPromptBaseline_ValidateEnterBeforeEdit(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace host",
		Validate: func(s string) error {
			if s == "" {
				return errors.New("host is required")
			}
			return nil
		},
	})
	tm.WaitFor("Workspace host")
	tm.Type(termtest.KeyEnter)
	tm.Golden("01-enter")

	tm.Type(termtest.KeyCtrlC)
	_, err := tm.Result()
	require.Error(t, err)
	assert.EqualError(t, err, "^C")
}

// TestPromptBaseline_ValidateValidBeforeEdit pins that an untouched prompt
// whose input already passes Validate shows "✔".
func TestPromptBaseline_ValidateValidBeforeEdit(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace host",
		Validate: func(s string) error {
			if strings.Contains(s, " ") {
				return errors.New("must not contain spaces")
			}
			return nil
		},
	})
	tm.WaitFor("Workspace host")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyEnter)
	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Empty(t, v)
}

// TestPromptBaseline_ValidateNonEditKeysKeepBlankGlyph pins that keys which
// leave the buffer unchanged do not flip the blank glyph to "✗".
func TestPromptBaseline_ValidateNonEditKeysKeepBlankGlyph(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace host",
		Validate: func(s string) error {
			if s == "" {
				return errors.New("host is required")
			}
			return nil
		},
	})
	tm.WaitFor("Workspace host")
	tm.Type(termtest.KeyLeft + termtest.KeyRight + termtest.KeyBackspace + termtest.KeyCtrlC)
	_, err := tm.Result()
	assert.EqualError(t, err, "^C")
	// The prompt line is erased on exit, so check every rendered frame.
	assert.NotContains(t, tm.Raw(), "✗")
}

// TestPromptBaseline_ValidateEmptyPasteKeepsBlankGlyph pins that an empty
// bracketed paste, which some terminals send for an empty clipboard, is not
// treated as an edit.
func TestPromptBaseline_ValidateEmptyPasteKeepsBlankGlyph(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace host",
		Validate: func(s string) error {
			if s == "" {
				return errors.New("host is required")
			}
			return nil
		},
	})
	tm.WaitFor("Workspace host")
	tm.Type("\x1b[200~\x1b[201~" + termtest.KeyCtrlC)
	_, err := tm.Result()
	assert.EqualError(t, err, "^C")
	assert.NotContains(t, tm.Raw(), "✗")
}
