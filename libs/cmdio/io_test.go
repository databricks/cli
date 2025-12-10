package cmdio

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGitBash(t *testing.T) {
	ctx := context.Background()
	assert.False(t, isGitBash(ctx))

	ctx = env.Set(ctx, "MSYSTEM", "MINGW64")
	ctx = env.Set(ctx, "TERM", "xterm")
	ctx = env.Set(ctx, "PS1", "\\[\033]0;$TITLEPREFIX:$PWD\007\\]\n\\[\033[32m\\]\\u@\\h \\[\033[35m\\]$MSYSTEM \\[\033[33m\\]\\w\\[\033[36m\\]`__git_ps1`\\[\033[0m\\]\n$")
	assert.True(t, isGitBash(ctx))
}

func TestCoordinatedWriter(t *testing.T) {
	var buf bytes.Buffer
	w := newCoordinatedWriter(&buf)
	defer w.close()

	// Test simple write
	n, err := w.Write([]byte("test message\n"))
	require.NoError(t, err)
	assert.Equal(t, 13, n)

	// Allow coordinator goroutine to process
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, "test message\n", buf.String())
}

func TestCoordinatedWriterConcurrent(t *testing.T) {
	var buf bytes.Buffer
	w := newCoordinatedWriter(&buf)
	defer w.close()

	// Write concurrently from multiple goroutines
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			msg := []byte("message\n")
			_, err := w.Write(msg)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Allow coordinator goroutine to process all writes
	time.Sleep(50 * time.Millisecond)

	// All messages should be written
	assert.Equal(t, 10, bytes.Count(buf.Bytes(), []byte("message\n")))
}

func TestCoordinatedWriterBeforeCmdIOInitialized(t *testing.T) {
	// Test that CoordinatedWriter returns a fallback writer when cmdIO
	// is not yet in the context (e.g., during early logger initialization)
	ctx := context.Background()
	w := CoordinatedWriter(ctx)
	require.NotNil(t, w)
	// Should return os.Stderr as fallback, not panic
}
