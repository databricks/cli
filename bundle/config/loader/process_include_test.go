package loader_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessInclude(t *testing.T) {
	b := &bundle.Bundle{
		RootPath: "testdata",
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "foo",
			},
		},
	}

	m := loader.ProcessInclude(filepath.Join(b.RootPath, "host.yml"), "host.yml")
	assert.Equal(t, "ProcessInclude(host.yml)", m.Name())

	// Assert the host value prior to applying the mutator
	assert.Equal(t, "foo", b.Config.Workspace.Host)

	// Apply the mutator and assert that the host value has been updated
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}
