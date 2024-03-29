package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestLocation(t *testing.T) {
	loc := dyn.Location{File: "file", Line: 1, Column: 2}
	assert.Equal(t, "file:1:2", loc.String())
}

func TestLocationDirectory(t *testing.T) {
	loc := dyn.Location{File: "file", Line: 1, Column: 2}
	dir, err := loc.Directory()
	assert.NoError(t, err)
	assert.Equal(t, ".", dir)
}

func TestLocationDirectoryNoFile(t *testing.T) {
	loc := dyn.Location{}
	_, err := loc.Directory()
	assert.Error(t, err)
}
