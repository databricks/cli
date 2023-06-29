package internal

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
)

func TestAccGitClonePrivateRepository(t *testing.T) {
	tmpDir := t.TempDir()

	// This is a private repository only accessible to databricks employees
	err := git.Clone("databricks", "bundle-samples-internal", tmpDir)
	assert.NoError(t, err)

	// assert examples from the private repository
	assert.DirExists(t, filepath.Join(tmpDir, "shark_sightings"))
	assert.DirExists(t, filepath.Join(tmpDir, "wikipedia_clickstream"))
}
