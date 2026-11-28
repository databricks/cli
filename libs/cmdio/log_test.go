package cmdio_test

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
)

func TestLogProgressWritesWhenNotQuiet(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

	cmdio.LogProgress(ctx, "Uploading bundle files")

	assert.Contains(t, stderr.String(), "Uploading bundle files")
}

func TestLogProgressSuppressedWhenQuiet(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = cmdio.WithQuiet(ctx)

	cmdio.LogProgress(ctx, "Uploading bundle files")

	assert.Empty(t, stderr.String())
}

// LogString is for results the user asked for, so -qq must not silence it.
func TestLogStringNotAffectedByQuiet(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = cmdio.WithQuiet(ctx)

	cmdio.LogString(ctx, "Resources: 1 created")

	assert.Contains(t, stderr.String(), "Resources: 1 created")
}

// Callers style LogString input for stdout (cmdio.Yellow checks stdout), so a
// stderr without color must still get plain text.
func TestLogStringStripsStylesOnColorlessStderr(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

	cmdio.LogString(ctx, "\x1b[33mhost key changed\x1b[0m: "+cmdio.Hyperlink("https://example.test", "docs"))

	assert.Equal(t, "host key changed: docs\n", stderr.String())
}

// lockedBuffer is a bytes.Buffer safe for concurrent writes, standing in for
// a stderr that is.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// The stderr writer is built once per cmdIO; concurrent first calls must
// agree on it (run with -race).
func TestLogStringConcurrent(t *testing.T) {
	stderr := &lockedBuffer{}
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, nil, io.Discard, stderr, "", ""))

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { cmdio.LogString(ctx, "\x1b[33mline\x1b[0m") })
	}
	wg.Wait()

	assert.Equal(t, strings.Repeat("line\n", 8), stderr.String())
}

func TestIsQuiet(t *testing.T) {
	ctx := t.Context()
	assert.False(t, cmdio.IsQuiet(ctx))
	assert.True(t, cmdio.IsQuiet(cmdio.WithQuiet(ctx)))
}
