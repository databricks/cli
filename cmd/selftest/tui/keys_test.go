package tui

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRawKeysPrintsChunksAndExitsOnDoubleCtrlC(t *testing.T) {
	// The pipe stays open after the data, so EOF can't end the call: only the
	// second Ctrl+C can.
	r, w := io.Pipe()
	t.Cleanup(func() { _ = w.Close() })
	go func() { _, _ = w.Write([]byte("a\x03\x03b")) }()

	var out bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- runRawKeys(r, &out, 0) }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("runRawKeys did not return after two Ctrl+C")
	}
	assert.Equal(t, "raw bytes=61 03 03 62 text=\"a\\x03\\x03b\"\n", out.String())
}

func TestRunRawKeysReturnsOnEOF(t *testing.T) {
	var out bytes.Buffer
	err := runRawKeys(bytes.NewReader([]byte("hi")), &out, 0)
	require.NoError(t, err)
	assert.Equal(t, "raw bytes=68 69 text=\"hi\"\n", out.String())
}
