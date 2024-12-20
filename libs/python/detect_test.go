package python

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectVEnvExecutable(t *testing.T) {
	dir := t.TempDir()
	interpreterPath := interpreterPath(dir)

	err := os.Mkdir(filepath.Dir(interpreterPath), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(interpreterPath, []byte(""), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "pyvenv.cfg"), []byte(""), 0o755)
	require.NoError(t, err)

	executable, err := DetectVEnvExecutable(dir)

	assert.NoError(t, err)
	assert.Equal(t, interpreterPath, executable)
}

func TestDetectVEnvExecutable_badLayout(t *testing.T) {
	dir := t.TempDir()

	_, err := DetectVEnvExecutable(dir)

	assert.Errorf(t, err, "can't find %q, check if virtualenv is created", interpreterPath(dir))
}

func interpreterPath(venvPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	} else {
		return filepath.Join(venvPath, "bin", "python3")
	}
}
