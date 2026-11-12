package aitools

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBinaryPathForAbsoluteArg0KeepsSymlinkPath(t *testing.T) {
	// The invoked path is kept as is, so a Homebrew symlink is not replaced by its target.
	arg0 := filepath.Join(t.TempDir(), "bin", "databricks")
	lookPath := func(string) (string, error) {
		t.Fatal("PATH lookup must not run when arg0 is absolute")
		return "", nil
	}

	path, absolute := binaryPathFor(arg0, lookPath)
	assert.True(t, absolute)
	assert.Equal(t, arg0, path)
}

func TestBinaryPathForRelativeArg0UsesPathLookup(t *testing.T) {
	found := filepath.Join(t.TempDir(), "databricks")
	var looked string
	lookPath := func(name string) (string, error) {
		looked = name
		return found, nil
	}

	path, absolute := binaryPathFor("./databricks", lookPath)
	assert.True(t, absolute)
	assert.Equal(t, found, path)
	assert.Equal(t, "databricks", looked)
}

func TestBinaryPathForFallsBackToBareName(t *testing.T) {
	notFound := func(string) (string, error) { return "", errors.New("not found") }
	path, absolute := binaryPathFor("databricks", notFound)
	assert.False(t, absolute)
	assert.Equal(t, "databricks", path)
}

func TestBinaryPathForRelativeLookupResultFallsBackToBareName(t *testing.T) {
	// A relative PATH entry does not give a path an agent can start from another directory.
	relative := func(string) (string, error) { return filepath.Join("bin", "databricks"), nil }
	path, absolute := binaryPathFor("databricks", relative)
	assert.False(t, absolute)
	assert.Equal(t, "databricks", path)
}
