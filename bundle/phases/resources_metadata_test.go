package phases

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDirectStateSizes_GroupsByResourceType(t *testing.T) {
	db := dstate.Database{
		State: map[string]dstate.ResourceEntry{
			"resources.jobs.foo":      {State: json.RawMessage(`{"name":"foo","x":1}`)},  // 20
			"resources.jobs.bar":      {State: json.RawMessage(`{"n":"bar"}`)},           // 11
			"resources.jobs.baz":      {State: json.RawMessage(`{"name":"baz","y":42}`)}, // 21
			"resources.pipelines.qux": {State: json.RawMessage(`{"name":"qux"}`)},        // 14
		},
	}
	raw, err := json.Marshal(db)
	require.NoError(t, err)

	got := parseDirectStateSizes(t.Context(), raw)
	assert.ElementsMatch(t, []int64{20, 11, 21}, got["jobs"])
	assert.ElementsMatch(t, []int64{14}, got["pipelines"])
}

func TestParseDirectStateSizes_SubResources(t *testing.T) {
	db := dstate.Database{
		State: map[string]dstate.ResourceEntry{
			"resources.jobs.foo":             {State: json.RawMessage(`{}`)},
			"resources.jobs.foo.permissions": {State: json.RawMessage(`[]`)},
		},
	}
	raw, _ := json.Marshal(db)
	got := parseDirectStateSizes(t.Context(), raw)
	assert.Len(t, got["jobs"], 1)
	assert.Len(t, got["permissions"], 1)
}

func TestParseDirectStateSizes_Malformed(t *testing.T) {
	assert.Nil(t, parseDirectStateSizes(t.Context(), []byte("not json")))
}

func TestParseTerraformStateSizes_TranslatesAndGroups(t *testing.T) {
	tfstate := map[string]any{
		"version": 4,
		"resources": []any{
			map[string]any{
				"type": "databricks_job",
				"mode": "managed",
				"instances": []any{
					map[string]any{"attributes": map[string]any{"id": "1001", "name": "foo"}},
					map[string]any{"attributes": map[string]any{"id": "1002", "name": "bar"}},
				},
			},
			map[string]any{
				"type": "databricks_pipeline",
				"mode": "managed",
				"instances": []any{
					map[string]any{"attributes": map[string]any{"id": "abc"}},
				},
			},
			map[string]any{
				"type": "databricks_unknown_provider_type",
				"mode": "managed",
				"instances": []any{
					map[string]any{"attributes": map[string]any{"id": "xyz"}},
				},
			},
		},
	}
	raw, err := json.Marshal(tfstate)
	require.NoError(t, err)

	got := parseTerraformStateSizes(t.Context(), raw)
	assert.Len(t, got["jobs"], 2)
	assert.Len(t, got["pipelines"], 1)

	for k := range got {
		assert.NotContains(t, k, "unknown")
	}
}

func TestParseTerraformStateSizes_SkipsDataSources(t *testing.T) {
	tfstate := map[string]any{
		"version": 4,
		"resources": []any{
			map[string]any{
				"type": "databricks_job",
				"mode": "data",
				"instances": []any{
					map[string]any{"attributes": map[string]any{"id": "1001"}},
				},
			},
		},
	}
	raw, _ := json.Marshal(tfstate)
	got := parseTerraformStateSizes(t.Context(), raw)
	assert.Empty(t, got["jobs"])
}

func TestParseTerraformStateSizes_Malformed(t *testing.T) {
	assert.Nil(t, parseTerraformStateSizes(t.Context(), []byte("not json")))
}

func TestResourceTypeFromKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"resources.jobs.foo", "jobs"},
		{"resources.pipelines.bar", "pipelines"},
		{"resources.jobs.foo.permissions", "permissions"},
		{"resources.secret_scopes.s.permissions", "permissions"},
		{"not-a-state-key", ""},
		{"resources.jobs", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, resourceTypeFromKey(c.in), "key=%q", c.in)
	}
}

func TestReadStateFile_MissingReturnsNilNil(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "missing.json")
	raw, err := readStateFile(tmp)
	assert.NoError(t, err)
	assert.Nil(t, raw)
}

func TestReadStateFile_ReadsExistingFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(tmp, []byte("hello"), 0o600))
	raw, err := readStateFile(tmp)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), raw)
}

func TestStatHelpers(t *testing.T) {
	// Helpers expect a sorted slice (collectResourcesMetadata sorts before calling).
	assert.Equal(t, int64(3), statMax([]int64{1, 2, 3}))
	assert.Equal(t, int64(2), statMean([]int64{1, 2, 3}))
	assert.Equal(t, int64(2), statMedian([]int64{1, 2, 3}))
	// Lower-middle for even count: sorted [1,2,3,4] -> index (4-1)/2 = 1 -> 2.
	assert.Equal(t, int64(2), statMedian([]int64{1, 2, 3, 4}))
	// Empty slices are zero.
	assert.Equal(t, int64(0), statMax(nil))
	assert.Equal(t, int64(0), statMean(nil))
	assert.Equal(t, int64(0), statMedian(nil))
}

func TestUnionKeys(t *testing.T) {
	got := unionKeys(map[string]int64{"a": 1, "b": 2}, map[string][]int64{"b": nil, "c": nil})
	assert.ElementsMatch(t, []string{"a", "b", "c"}, got)
}
