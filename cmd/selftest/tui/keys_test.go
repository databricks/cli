package tui

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRawKeysPrintsChunksAndExitsOnDoubleCtrlC(t *testing.T) {
	var out bytes.Buffer
	err := runRawKeys(bytes.NewReader([]byte("a\x03\x03b")), &out, 0)
	require.NoError(t, err)
	assert.Equal(t, "raw bytes=61 03 03 62 text=\"a\\x03\\x03b\"\n", out.String())
}

func TestRunRawKeysReturnsOnEOF(t *testing.T) {
	var out bytes.Buffer
	err := runRawKeys(bytes.NewReader([]byte("hi")), &out, 0)
	require.NoError(t, err)
	assert.Equal(t, "raw bytes=68 69 text=\"hi\"\n", out.String())
}
