package cmdiotest_test

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_UTF8 pins multi-byte rune handling: typing "café"
// (4 runes, 5 bytes) renders as 4 cells, one Backspace deletes one rune
// not one byte, and the returned value preserves the original code points.
// An implementation that counts bytes instead of runes would silently
// corrupt non-ASCII input even with ASCII tests passing.
func TestPromptBaseline_UTF8(t *testing.T) {
	// On Windows, bubbletea wraps non-console input in
	// github.com/mattn/go-localereader (see key_windows.go), which decodes
	// each incoming byte ≥0x80 as the system ANSI code page (CP1252 on
	// English Windows). Our pipe-based harness feeds raw UTF-8, so the
	// c3 a9 bytes for "é" get re-read as Latin-1 Ã + © and never reach
	// the prompt model as a single rune. The model itself handles UTF-8
	// correctly in production (where bytes come from the real console).
	if runtime.GOOS == "windows" {
		t.Skip("bubbletea localereader mangles UTF-8 over non-console input on Windows")
	}
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Name",
	})
	tm.WaitFor("Name")
	tm.Golden("01-empty")

	tm.Type("café")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Golden("03-after-backspace")

	tm.Type("é")
	tm.Golden("04-restored")

	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "café", v, "snapshot:\n%s", tm.Snapshot())
}
