package dynpath

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
)

func TestConvertPathNodeToDynPath(t *testing.T) {
	tests := []struct {
		name       string
		structPath string       // PathNode string representation to be parsed
		dynPath    string       // Expected DynPath format
		rootType   reflect.Type // optional, for context-aware wildcard rendering
	}{
		// Basic node types
		{
			name:       "nil path",
			structPath: "",
			dynPath:    "",
		},
		{
			name:       "array index",
			structPath: "[5]",
			dynPath:    "[5]",
		},
		{
			name:       "map key",
			structPath: "['mykey']",
			dynPath:    "mykey",
		},
		{
			name:       "struct field",
			structPath: "json_name",
			dynPath:    "json_name",
		},
		{
			name:       "dot star",
			structPath: "*",
			dynPath:    "*",
		},
		{
			name:       "bracket star - array type",
			structPath: "[*]",
			rootType:   reflect.TypeOf([]int{}),
			dynPath:    "[*]",
		},
		{
			name:       "bracket star - map type",
			structPath: "[*]",
			rootType:   reflect.TypeOf(map[string]int{}),
			dynPath:    "*",
		},

		// Compound paths
		{
			name:       "struct field -> array index",
			structPath: "items[3]",
			dynPath:    "items[3]",
		},
		{
			name:       "struct field -> map key",
			structPath: "config['database']",
			dynPath:    "config.database",
		},
		{
			name:       "struct field -> struct field",
			structPath: "user.name",
			dynPath:    "user.name",
		},
		{
			name:       "map key -> array index",
			structPath: "['servers'][0]",
			dynPath:    "servers[0]",
		},
		{
			name:       "map key -> struct field",
			structPath: "['primary'].host",
			dynPath:    "primary.host",
		},
		{
			name:       "array index -> struct field",
			structPath: "[2].id",
			dynPath:    "[2].id",
		},
		{
			name:       "array index -> map key",
			structPath: "[1]['status']",
			dynPath:    "[1].status",
		},

		// Wildcard combinations
		{
			name:       "dot star with parent",
			structPath: "Parent.*",
			dynPath:    "Parent.*",
		},
		{
			name:       "bracket star with parent - array",
			structPath: "Parent[*]",
			rootType: reflect.TypeOf(struct {
				Parent []int
			}{}),
			dynPath: "Parent[*]",
		},

		{
			name:       "bracket star with parent - array",
			structPath: "parent[*]",
			rootType: reflect.TypeOf(struct {
				Parent []int `json:"parent"`
			}{}),
			dynPath: "parent[*]",
		},

		{
			name:       "bracket star with parent - map",
			structPath: "Parent[*]",
			rootType: reflect.TypeOf(struct {
				Parent map[string]int
			}{}),
			dynPath: "Parent.*",
		},

		// Special characters and edge cases
		{
			name:       "map key with single quote",
			structPath: "['key''s']",
			dynPath:    "key's",
		},
		{
			name:       "map key with multiple single quotes",
			structPath: "['''''']",
			dynPath:    "''",
		},
		{
			name:       "empty map key",
			structPath: "['']",
			dynPath:    "",
		},
		{
			name:       "map key with reserved characters",
			structPath: "['key\x00[],`']",
			dynPath:    "key\x00[],`",
		},
		{
			name:       "field with special characters",
			structPath: "field@name:with#symbols!",
			dynPath:    "field@name:with#symbols!",
		},
		{
			name:       "field with spaces",
			structPath: "field with spaces",
			dynPath:    "field with spaces",
		},
		{
			name:       "field with unicode",
			structPath: "名前🙂",
			dynPath:    "名前🙂",
		},

		// Complex real-world example
		{
			name:       "complex nested path",
			structPath: "user.*.settings['theme'][0].color",
			dynPath:    "user.*.settings.theme[0].color",
		},

		// Mixed JSON tag and Go field name scenarios
		//{
		//	name:       "Go field -> JSON field",
		//	structPath: "Parent.child_name",
		//	dynPath:    "Parent.child_name",
		//},
		//{
		//	name:       "JSON field -> Go field",
		//	structPath: "parent.ChildName",
		//	dynPath:    "parent.ChildName",
		//},
		//{
		//	name:       "dash JSON tag field",
		//	structPath: "-",
		//	dynPath:    "-",
		//},
		//{
		//	name:       "JSON tag with options",
		//	structPath: "lazy_field",
		//	dynPath:    "lazy_field",
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathNode, err := structpath.Parse(tt.structPath)
			t.Logf("%q node=%#v", tt.structPath, pathNode)
			assert.NoError(t, err, "Failed to parse structPath: %s", tt.structPath)

			// Convert to DynPath and verify result
			result := ConvertPathNodeToDynPath(pathNode, tt.rootType)
			assert.Equal(t, tt.dynPath, result, "ConvertPathNodeToDynPath conversion should match expected DynPath format")
		})
	}
}
