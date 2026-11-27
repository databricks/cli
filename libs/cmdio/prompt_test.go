package cmdio

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptModelNoColor(t *testing.T) {
	m := newPromptModel(PromptOptions{Label: "Name"}, true)
	assert.Contains(t, m.View(), ansiGreen)

	m = newPromptModel(PromptOptions{Label: "Name"}, false)
	v := m.View()
	assert.NotContains(t, v, ansiGreen)
	assert.Contains(t, v, ansiBold)
}

// syncBuffer is a bytes.Buffer that is safe to read while bubbletea writes.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// runWithKeys runs fn against a fully interactive test cmdIO and types keys
// once the first frame containing label is on screen, so assertions about
// that frame don't depend on timing. It returns what was rendered to stderr.
func runWithKeys(t *testing.T, noColor bool, label, keys string, fn func(c *cmdIO) error) string {
	t.Helper()
	in, w := io.Pipe()
	out := &syncBuffer{}
	c := NewTestIO(in, io.Discard, out)
	c.capabilities.noColor = noColor

	go func() {
		assert.Eventually(t, func() bool { return strings.Contains(out.String(), label) }, 5*time.Second, 10*time.Millisecond)
		_, _ = w.Write([]byte(keys))
	}()
	require.NoError(t, fn(c))
	_ = w.Close()
	return out.String()
}

func TestRunPromptNoColor(t *testing.T) {
	run := func(c *cmdIO) error {
		v, err := RunPrompt(InContext(t.Context(), c), PromptOptions{Label: "Name"})
		assert.Equal(t, "x", v)
		return err
	}

	assert.Contains(t, runWithKeys(t, false, "Name", "x\r", run), ansiGreen)

	out := runWithKeys(t, true, "Name", "x\r", run)
	assert.NotContains(t, out, ansiGreen)
	assert.Contains(t, out, ansiBold)
}
