package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitClonePublicRepository(t *testing.T) {
	tmpDir := t.TempDir()
	var err error

	err = Clone("databricks", "cli", tmpDir)
	assert.NoError(t, err)
	assert.DirExists(t, filepath.Join(tmpDir, "cli-main"))

	b, err := os.ReadFile(filepath.Join(tmpDir, "cli-main/NOTICE"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "Copyright (2023) Databricks, Inc.")
}
