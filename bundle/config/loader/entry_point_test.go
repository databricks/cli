package loader_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryPointNoRootPath(t *testing.T) {
	b := &bundle.Bundle{}
	diags := bundle.Apply(t.Context(), b, loader.EntryPoint())
	require.Error(t, diags.Error())
}

func TestEntryPoint(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "testdata/basic",
	}
	diags := bundle.Apply(t.Context(), b, loader.EntryPoint())
	require.NoError(t, diags.Error())
	assert.Equal(t, "loader_test", b.Config.Bundle.Name)
}
