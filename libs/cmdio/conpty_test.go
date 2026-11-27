package cmdio

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMSYSPipeNotAFile(t *testing.T) {
	assert.False(t, isMSYSPipe(&bytes.Buffer{}))
	assert.False(t, isMSYSPipe(FakeTTY(&bytes.Buffer{})))
	assert.False(t, isMSYSPipe(nil))
}

func TestIsMSYSPipeRegularFile(t *testing.T) {
	f, err := os.Create(filepath.Join(t.TempDir(), "stdin"))
	require.NoError(t, err)
	defer f.Close()
	assert.False(t, isMSYSPipe(f))
}

func TestIsMSYSPipeDevNull(t *testing.T) {
	f, err := os.Open(os.DevNull)
	require.NoError(t, err)
	defer f.Close()
	assert.False(t, isMSYSPipe(f))
}

func TestIsMSYSPipeAnonymousPipe(t *testing.T) {
	// A plain pipe (e.g. `echo x | databricks ...`) is not an MSYS2 pty pipe;
	// it is rejected for prompts because it is not a TTY at all.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()
	assert.False(t, isMSYSPipe(r))
	assert.False(t, isTTY(r))
}

func TestMSYSPipeBlocksPromptsOnly(t *testing.T) {
	// An MSYS2 pty pipe counts as a TTY for output and line input, but not
	// for prompts and pagers, which need raw keystrokes.
	c := Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true, stdinIsMSYSPipe: true}
	assert.True(t, c.SupportsInteractive())
	assert.True(t, c.SupportsColor(FakeTTY(&bytes.Buffer{})))
	assert.False(t, c.SupportsPrompt())
	assert.False(t, c.SupportsPager())
	assert.Equal(t, InteractiveModeOutputOnly, c.InteractiveMode())
}
