package cmdio

import (
	"bytes"
	"io"
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

// runWithKeys runs fn against a fully interactive test cmdIO and types keys
// once it is running. It returns what was rendered to stderr.
func runWithKeys(t *testing.T, noColor bool, keys string, fn func(c *cmdIO) error) string {
	t.Helper()
	in, w := io.Pipe()
	var out bytes.Buffer
	c := NewTestIO(in, io.Discard, &out)
	c.capabilities.noColor = noColor

	go func() {
		time.Sleep(200 * time.Millisecond)
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

	assert.Contains(t, runWithKeys(t, false, "x\r", run), ansiGreen)

	out := runWithKeys(t, true, "x\r", run)
	assert.NotContains(t, out, ansiGreen)
	assert.Contains(t, out, ansiBold)
}
