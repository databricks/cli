package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynPathToStructPath(t *testing.T) {
	tests := []struct {
		path     dyn.Path
		expected string
	}{
		{
			path:     dyn.NewPath(dyn.Key("foo"), dyn.Key("bar")),
			expected: "foo.bar",
		},
		{
			path:     dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar")),
			expected: "foo[1].bar",
		},
		{
			path:     dyn.NewPath(dyn.Key("configuration"), dyn.Key("europris.swipe.egress_streaming_schema")),
			expected: "configuration['europris.swipe.egress_streaming_schema']",
		},
		{
			path:     dyn.NewPath(dyn.Key("tags"), dyn.Key("it's.here")),
			expected: "tags['it''s.here']",
		},
	}

	for _, tc := range tests {
		node := dynPathToStructPath(tc.path)
		assert.Equal(t, tc.expected, node.String())
	}
}

// TestMakePlanRejectsVariableInResourceKey verifies that a variable reference
// in a resource map key (e.g. resources.schemas.${var.schema}) is rejected
// with a clear error rather than panicking with a nil pointer dereference
// inside PrepareState. Regression test for issue #5098.
func TestMakePlanRejectsVariableInResourceKey(t *testing.T) {
	rootCfg := config.Root{
		Resources: config.Resources{
			Schemas: map[string]*resources.Schema{
				"${var.schema}": {},
			},
		},
	}
	require.NoError(t, rootCfg.Mutate(func(v dyn.Value) (dyn.Value, error) { return v, nil }))

	db := dstate.NewDatabase("", 0)
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)
	b := &DeploymentBundle{Adapters: adapters}

	_, err = b.makePlan(t.Context(), &rootCfg, &db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "${var.schema}")
	assert.Contains(t, err.Error(), "cannot contain variable references")
}
