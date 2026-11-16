//go:build unix

package aircmd

import (
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// git lists neither FIFOs nor sockets, and opening a FIFO to package it would block.
func TestSnapshotFilesWithoutGit_SkipsFIFO(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, "a.py", "1")
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "pipe"), 0o600))

	assert.Equal(t, []string{"a.py"}, selectedNames(t, dir, nil, false))
	withoutGit(t)
	assert.Equal(t, []string{"a.py"}, selectedNames(t, dir, nil, false))
}
