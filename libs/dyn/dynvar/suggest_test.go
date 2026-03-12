package dynvar

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b     string
		expected int
	}{
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"ab", "abc", 1},
		{"abc", "axc", 1},
		{"kitten", "sitting", 3},
		{"", "abc", 3},
		{"abc", "", 3},
		{"", "", 0},
	}

	for _, tc := range cases {
		t.Run(tc.a+"_"+tc.b, func(t *testing.T) {
			assert.Equal(t, tc.expected, levenshtein(tc.a, tc.b))
		})
	}
}

func TestClosestKeyMatch(t *testing.T) {
	cases := []struct {
		name         string
		key          string
		candidates   []string
		expectedKey  string
		expectedDist int
	}{
		{
			name:         "close match",
			key:          "nme",
			candidates:   []string{"name", "type", "value"},
			expectedKey:  "name",
			expectedDist: 1,
		},
		{
			name:         "no match",
			key:          "xyz",
			candidates:   []string{"name", "type"},
			expectedKey:  "",
			expectedDist: 0,
		},
		{
			name:         "exact match",
			key:          "name",
			candidates:   []string{"name", "type"},
			expectedKey:  "name",
			expectedDist: 0,
		},
		{
			name:         "threshold boundary within",
			key:          "ab",
			candidates:   []string{"ax"},
			expectedKey:  "ax",
			expectedDist: 1,
		},
		{
			name:         "threshold boundary exceeds",
			key:          "ab",
			candidates:   []string{"xyz"},
			expectedKey:  "",
			expectedDist: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			match, dist := closestKeyMatch(tc.key, tc.candidates)
			assert.Equal(t, tc.expectedKey, match)
			assert.Equal(t, tc.expectedDist, dist)
		})
	}
}

func TestSuggestPath(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("my-bundle"),
			"git": dyn.V(map[string]dyn.Value{
				"branch": dyn.V("main"),
			}),
		}),
		"variables": dyn.V(map[string]dyn.Value{
			"my_cluster_id": dyn.V("abc123"),
		}),
		"workspace": dyn.V(map[string]dyn.Value{
			"file_path": dyn.V("/path"),
		}),
	})

	cases := []struct {
		name     string
		root     dyn.Value
		path     dyn.Path
		expected string
	}{
		{
			name:     "single typo",
			root:     root,
			path:     dyn.MustPathFromString("bundle.nme"),
			expected: "bundle.name",
		},
		{
			name:     "multi-level typo",
			root:     root,
			path:     dyn.MustPathFromString("bundel.nme"),
			expected: "bundle.name",
		},
		{
			name:     "nested",
			root:     root,
			path:     dyn.MustPathFromString("bundle.git.brnch"),
			expected: "bundle.git.branch",
		},
		{
			name:     "variable typo",
			root:     root,
			path:     dyn.MustPathFromString("variables.my_clster_id"),
			expected: "variables.my_cluster_id",
		},
		{
			name:     "no match",
			root:     root,
			path:     dyn.MustPathFromString("variables.completely_wrong_name_that_is_very_different"),
			expected: "",
		},
		{
			name:     "exact match",
			root:     root,
			path:     dyn.MustPathFromString("bundle.name"),
			expected: "bundle.name",
		},
		{
			name:     "root key does not exist",
			root:     root,
			path:     dyn.MustPathFromString("nonexistent.field"),
			expected: "",
		},
		{
			name:     "path into non-map",
			root:     root,
			path:     dyn.MustPathFromString("bundle.name.sub"),
			expected: "",
		},
		{
			name: "index passthrough",
			root: dyn.V(map[string]dyn.Value{
				"items": dyn.V([]dyn.Value{
					dyn.V(map[string]dyn.Value{
						"value": dyn.V("first"),
					}),
				}),
			}),
			path:     dyn.NewPath(dyn.Key("items"), dyn.Index(0), dyn.Key("vlue")),
			expected: "items[0].value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, SuggestPath(tc.root, tc.path))
		})
	}
}
