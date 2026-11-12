package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigFilesCachesAFailedRead checks that a config file which cannot be read is read once per lookup: later
// profiles of the same file get the same failure, without a second read that could differ.
func TestConfigFilesCachesAFailedRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte("[broken\n"), 0o600))

	files := configFiles{}
	_, first := files.profile(path, "logfood")
	require.Error(t, first)

	require.NoError(t, os.WriteFile(path, []byte("[logfood]\nhost = https://adb-1.test\n"), 0o600))
	_, second := files.profile(path, "logfood")
	assert.Equal(t, first, second, "the failure of the file is kept for the lookup")
}
