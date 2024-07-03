package vfs

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilerPath(t *testing.T) {
	ctx := context.Background()
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Create a new filer-backed path.
	p, err := NewFilerPath(ctx, filepath.FromSlash(wd), filer.NewLocalClient)
	require.NoError(t, err)

	// Open self.
	f, err := p.Open("filer_test.go")
	require.NoError(t, err)
	defer f.Close()

	// Run stat on self.
	s, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, "filer_test.go", s.Name())
	assert.GreaterOrEqual(t, int(s.Size()), 128)

	// Read some bytes.
	buf := make([]byte, 1024)
	_, err = f.Read(buf)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(buf), "package vfs\n"))

	// Open non-existent file.
	_, err = p.Open("doesntexist_test.go")
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Stat self.
	s, err = p.Stat("filer_test.go")
	require.NoError(t, err)
	assert.Equal(t, "filer_test.go", s.Name())
	assert.GreaterOrEqual(t, int(s.Size()), 128)

	// Stat non-existent file.
	_, err = p.Stat("doesntexist_test.go")
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// ReadDir self.
	entries, err := p.ReadDir(".")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1)

	// ReadDir non-existent directory.
	_, err = p.ReadDir("doesntexist")
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// ReadFile self.
	buf, err = p.ReadFile("filer_test.go")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(buf), "package vfs\n"))

	// ReadFile non-existent file.
	_, err = p.ReadFile("doesntexist_test.go")
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Parent self.
	pp := p.Parent()
	require.NotNil(t, pp)
	assert.Equal(t, path.Join(pp.Native(), "vfs"), p.Native())
}
