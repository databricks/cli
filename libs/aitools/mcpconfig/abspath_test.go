package mcpconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbsoluteAsWrittenKeepsAbsolutePaths(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "config.json")
	got, err := AbsoluteAsWritten(abs)
	require.NoError(t, err)
	assert.Equal(t, abs, got)
}

func TestAbsoluteAsWrittenJoinsRelativePathToWorkingDirectory(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	got, err := AbsoluteAsWritten("cfg/../.claude.json")
	require.NoError(t, err)
	// Not cleaned: the ".." stays, so that it resolves through a symlink as the OS does.
	assert.Equal(t, wd+string(filepath.Separator)+filepath.FromSlash("cfg/../.claude.json"), got)
}
