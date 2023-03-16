package flags

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogFileFlagDefault(t *testing.T) {
	f := NewLogFileFlag()
	assert.Equal(t, os.Stderr, f.Writer())
	assert.Equal(t, "stderr", f.String())
}

func TestLogFileFlagSetStdout(t *testing.T) {
	var err error

	f := NewLogFileFlag()
	err = f.Set("stdout")
	require.NoError(t, err)
	assert.Equal(t, os.Stdout, f.Writer())
	assert.Equal(t, "stdout", f.String())
	err = f.Set("STDOUT")
	require.NoError(t, err)
	assert.Equal(t, os.Stdout, f.Writer())
	assert.Equal(t, "stdout", f.String())
}

func TestLogFileFlagSetStderr(t *testing.T) {
	var err error

	f := NewLogFileFlag()
	err = f.Set("stderr")
	require.NoError(t, err)
	assert.Equal(t, os.Stderr, f.Writer())
	assert.Equal(t, "stderr", f.String())
	err = f.Set("STDERR")
	require.NoError(t, err)
	assert.Equal(t, os.Stderr, f.Writer())
	assert.Equal(t, "stderr", f.String())
}

func TestLogFileFlagSetNewFile(t *testing.T) {
	var err error

	// Synthesize path to logfile.
	dir := t.TempDir()
	path := filepath.Join(dir, "logfile")

	// Configure flag.
	f := NewLogFileFlag()
	err = f.Set(path)
	require.NoError(t, err)
	err = f.Open()
	require.NoError(t, err)
	defer f.Close()

	// Writer must be the underlying file.
	w := f.Writer()
	file, ok := w.(*os.File)
	require.True(t, ok)
	assert.Equal(t, path, file.Name())

	// String must be equal to the path.
	assert.Equal(t, path, f.String())
}

func TestLogFileFlagSetExistingFile(t *testing.T) {
	var err error

	// Synthesize path to logfile.
	dir := t.TempDir()
	path := filepath.Join(dir, "logfile")

	// Add some contents to temporary file.
	file, err := os.Create(path)
	require.NoError(t, err)
	_, err = file.WriteString("a\n")
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	// Configure flag.
	f := NewLogFileFlag()
	err = f.Set(path)
	require.NoError(t, err)
	err = f.Open()
	require.NoError(t, err)
	defer f.Close()

	// Writer must be the underlying file.
	w := f.Writer()
	file, ok := w.(*os.File)
	require.True(t, ok)
	assert.Equal(t, path, file.Name())

	// String must be equal to the path.
	assert.Equal(t, path, f.String())

	// Write more contents.
	_, err = w.Write([]byte("b\n"))
	require.NoError(t, err)

	// Verify that the contents was appended to the file.
	buf, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "a\nb\n", string(buf))
}

func TestLogFileFlagSetBadPath(t *testing.T) {
	var err error

	// Synthesize path that doesn't exist.
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid/logfile")

	// Configure flag.
	f := NewLogFileFlag()
	err = f.Set(path)
	require.NoError(t, err)
	err = f.Open()
	assert.Error(t, err)
}
