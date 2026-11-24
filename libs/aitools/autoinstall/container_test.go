package autoinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnyExists(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, ".dockerenv")
	missing := filepath.Join(dir, ".containerenv")

	assert.False(t, anyExists(nil))
	assert.False(t, anyExists([]string{missing}))

	require.NoError(t, os.WriteFile(marker, nil, 0o600))
	assert.True(t, anyExists([]string{missing, marker}))
}
