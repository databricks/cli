package direct

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
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

// extractReferences gates references on the state type: a reference in an input-only field
// (e.g. a bundle:"readonly" field like volumes' volume_path) must not become a dependency,
// while references in state fields (e.g. comment) are still extracted.
func TestExtractReferences_ExcludesReadonlyFields(t *testing.T) {
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)
	// The volume state type is catalog.CreateVolumeRequestContent, which has
	// comment but not volume_path.
	stateType := adapters["volumes"].StateType()

	const yml = `
resources:
  volumes:
    v:
      catalog_name: main
      schema_name: myschema
      name: myvol
      comment: "${resources.schemas.kept.name}"
      volume_path: "/Volumes/main/${resources.schemas.dropped.name}/myvol"
`
	root, err := yamlloader.LoadYAML("test", bytes.NewBufferString(yml))
	require.NoError(t, err)

	refs, err := extractReferences(root, "resources.volumes.v", stateType)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"comment": "${resources.schemas.kept.name}",
	}, refs)
}

func TestShouldSkipBackendDefault_ManagedPropertiesOnly(t *testing.T) {
	// Rules mirror the schemas backend_defaults in resources.yml, but the test is
	// deliberately self-contained so that edits to resources.yml don't break it.
	// The real wiring is covered by acceptance/bundle/resources/schemas/drift.
	rowTracking, err := structpath.ParsePattern("properties['unity.catalog.managed.delta.defaults.delta.enableRowTracking']")
	require.NoError(t, err)
	catalogManaged, err := structpath.ParsePattern("properties['unity.catalog.managed.iceberg.defaults.delta.feature.catalogManaged']")
	require.NoError(t, err)
	cfg := &dresources.ResourceLifecycleConfig{
		BackendDefaults: []dresources.BackendDefaultRule{
			{Field: rowTracking},
			{Field: catalogManaged},
		},
	}

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
