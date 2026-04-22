package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
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

func TestShouldSkipBackendDefault_SchemaManagedPropertiesOnly(t *testing.T) {
	cfg := dresources.GetResourceConfig("schemas")
	require.NotNil(t, cfg)

	tests := []struct {
		name     string
		path     string
		remote   any
		expected bool
	}{
		{
			name:     "managed delta row tracking property",
			path:     "properties['unity.catalog.managed.delta.defaults.delta.enableRowTracking']",
			remote:   "true",
			expected: true,
		},
		{
			name:     "managed iceberg catalog property",
			path:     "properties['unity.catalog.managed.iceberg.defaults.delta.feature.catalogManaged']",
			remote:   "true",
			expected: true,
		},
		{
			name:     "unmanaged remote-only property is not skipped",
			path:     "properties['custom.remote_only']",
			remote:   "true",
			expected: false,
		},
		{
			name:     "managed-only parent properties map is skipped",
			path:     "properties",
			remote:   map[string]string{"unity.catalog.managed.delta.defaults.delta.enableRowTracking": "true"},
			expected: true,
		},
		{
			name:     "mixed parent properties map is not skipped",
			path:     "properties",
			remote:   map[string]string{"unity.catalog.managed.delta.defaults.delta.enableRowTracking": "true", "custom.remote_only": "true"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)

			reason, ok := shouldSkipBackendDefault(cfg, path, &deployplan.ChangeDesc{
				Old:    nil,
				New:    nil,
				Remote: tt.remote,
			})

			assert.Equal(t, tt.expected, ok)
			if tt.expected {
				assert.Equal(t, deployplan.ReasonBackendDefault, reason)
			} else {
				assert.Empty(t, reason)
			}
		})
	}
}

// Map drift handling synthesizes child paths to match against rules. structdiff
// always emits map keys in bracket notation, so synthetic child paths must too;
// otherwise rules wouldn't match for identifier-like keys.
func TestShouldSkipBackendDefault_MapDriftUsesBracketKeys(t *testing.T) {
	field, err := structpath.ParsePattern("properties['simple']")
	require.NoError(t, err)
	cfg := &dresources.ResourceLifecycleConfig{
		BackendDefaults: []dresources.BackendDefaultRule{{Field: field}},
	}

	path, err := structpath.ParsePath("properties")
	require.NoError(t, err)

	reason, ok := shouldSkipBackendDefault(cfg, path, &deployplan.ChangeDesc{
		Remote: map[string]string{"simple": "v"},
	})
	assert.True(t, ok)
	assert.Equal(t, deployplan.ReasonBackendDefault, reason)
}
