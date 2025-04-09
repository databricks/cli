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
