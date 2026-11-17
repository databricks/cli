package cmdio

import (
	"bytes"
	"io"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptModelNoColor(t *testing.T) {
	m := newPromptModel(PromptOptions{Label: "Name"}, true)
	assert.Contains(t, m.render(), ansiGreen)

	m = newPromptModel(PromptOptions{Label: "Name"}, false)
	v := m.render()
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

	assert.True(t, hasSGR(runWithKeys(t, false, "Name", "x\r", run), ansiGreen))

	out := runWithKeys(t, true, "Name", "x\r", run)
	assert.False(t, hasSGR(out, ansiGreen))
	assert.True(t, hasSGR(out, ansiBold))
}

var sgrSequence = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// hasSGR reports whether out sets the attribute of the single-parameter SGR
// sequence want, e.g. "\x1b[32m". Bubble Tea v2 re-encodes what a view
// writes, merging parameters ("\x1b[32;1m"), so the bytes don't survive.
func hasSGR(out, want string) bool {
	code := sgrSequence.FindStringSubmatch(want)[1]
	for _, m := range sgrSequence.FindAllStringSubmatch(out, -1) {
		if slices.Contains(strings.Split(m[1], ";"), code) {
			return true
		}
	}
	return false
}
