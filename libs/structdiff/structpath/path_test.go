package structpath

import (
	"testing"

	"github.com/databricks/cli/libs/structdiff/jsontag"
	"github.com/stretchr/testify/assert"
)

func TestPathNode(t *testing.T) {
	tests := []struct {
		name    string
		node    *PathNode
		String  string
		DynPath string
		Index   any
		MapKey  any
		Field   any
		Root    any
	}{
		// Single node tests
		{
			name:   "nil path",
			node:   nil,
			String: "",
			Root:   true,
		},
		{
			name:   "array index",
			node:   NewIndex(nil, 5),
			String: "[5]",
			Index:  5,
		},
		{
			name:    "map key",
			node:    NewMapKey(nil, "mykey"),
			String:  `["mykey"]`,
			DynPath: "mykey",
			MapKey:  "mykey",
		},
		{
			name:    "struct field with JSON tag",
			node:    NewStructField(nil, jsontag.JSONTag("json_name"), "GoFieldName"),
			String:  ".json_name",
			DynPath: "json_name",
			Field:   "json_name",
		},
		{
			name:    "struct field without JSON tag (fallback to Go name)",
			node:    NewStructField(nil, jsontag.JSONTag(""), "GoFieldName"),
			String:  ".GoFieldName",
			DynPath: "GoFieldName",
			Field:   "GoFieldName",
		},
		{
			name:    "struct field with dash JSON tag",
			node:    NewStructField(nil, jsontag.JSONTag("-"), "GoFieldName"),
			String:  ".-",
			DynPath: "-",
			Field:   "-",
		},
		{
			name:    "struct field with JSON tag options",
			node:    NewStructField(nil, jsontag.JSONTag("lazy_field,omitempty"), "LazyField"),
			String:  ".lazy_field",
			DynPath: "lazy_field",
			Field:   "lazy_field",
		},
		{
			name:    "any key",
			node:    NewAnyKey(nil),
			String:  "[*]",
			DynPath: "*",
		},
		{
			name:   "any index",
			node:   NewAnyIndex(nil),
			String: "[*]",
		},

		// Two node tests
		{
			name:    "struct field -> array index",
			node:    NewIndex(NewStructField(nil, jsontag.JSONTag("items"), "Items"), 3),
			String:  ".items[3]",
			DynPath: "items[3]",
			Index:   3,
		},
		{
			name:    "struct field -> map key",
			node:    NewMapKey(NewStructField(nil, jsontag.JSONTag("config"), "Config"), "database"),
			String:  `.config["database"]`,
			DynPath: "config.database",
			MapKey:  "database",
		},
		{
			name:    "struct field -> struct field",
			node:    NewStructField(NewStructField(nil, jsontag.JSONTag("user"), "User"), jsontag.JSONTag("name"), "Name"),
			String:  ".user.name",
			DynPath: "user.name",
			Field:   "name",
		},
		{
			name:    "map key -> array index",
			node:    NewIndex(NewMapKey(nil, "servers"), 0),
			String:  `["servers"][0]`,
			DynPath: "servers[0]",
			Index:   0,
		},
		{
			name:    "map key -> struct field",
			node:    NewStructField(NewMapKey(nil, "primary"), jsontag.JSONTag("host"), "Host"),
			String:  `["primary"].host`,
			DynPath: `primary.host`,
			Field:   "host",
		},
		{
			name:   "array index -> struct field",
			node:   NewStructField(NewIndex(nil, 2), jsontag.JSONTag("id"), "ID"),
			String: "[2].id",
			Field:  "id",
		},
		{
			name:    "array index -> map key",
			node:    NewMapKey(NewIndex(nil, 1), "status"),
			String:  `[1]["status"]`,
			DynPath: "[1].status",
			MapKey:  "status",
		},
		{
			name:    "struct field without JSON tag -> struct field with JSON tag",
			node:    NewStructField(NewStructField(nil, jsontag.JSONTag(""), "Parent"), jsontag.JSONTag("child_name"), "ChildName"),
			String:  ".Parent.child_name",
			DynPath: "Parent.child_name",
			Field:   "child_name",
		},
		{
			name:    "any key",
			node:    NewAnyKey(NewStructField(nil, jsontag.JSONTag(""), "Parent")),
			String:  ".Parent[*]",
			DynPath: "Parent.*",
		},
		{
			name:    "any index",
			node:    NewAnyIndex(NewStructField(nil, jsontag.JSONTag(""), "Parent")),
			String:  ".Parent[*]",
			DynPath: "Parent[*]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test String() method
			result := tt.node.String()
			assert.Equal(t, tt.String, result, "String() method")

			// Test DynPath() method
			expectedDyn := tt.String
			if tt.DynPath != "" {
				expectedDyn = tt.DynPath
			}
			result = tt.node.DynPath()
			assert.Equal(t, expectedDyn, result, "DynPath() method")

			// Index
			gotIndex, isIndex := tt.node.Index()
			if tt.Index == nil {
				assert.Equal(t, -1, gotIndex)
				assert.False(t, isIndex)
			} else {
				expectedIndex := tt.Index.(int)
				assert.Equal(t, expectedIndex, gotIndex)
				assert.True(t, isIndex)
			}

			// Field
			gotField, isField := tt.node.Field()
			if tt.Field == nil {
				assert.Equal(t, "", gotField)
				assert.False(t, isField)
			} else {
				expected := tt.Field.(string)
				assert.Equal(t, expected, gotField)
				assert.True(t, isField)
			}

			// MapKey
			gotMapKey, isMapKey := tt.node.MapKey()
			if tt.MapKey == nil {
				assert.Equal(t, "", gotMapKey)
				assert.False(t, isMapKey)
			} else {
				expected := tt.MapKey.(string)
				assert.Equal(t, expected, gotMapKey)
				assert.True(t, isMapKey)
			}

			isRoot := tt.node.IsRoot()
			if tt.Root == nil {
				assert.False(t, isRoot)
			} else {
				assert.True(t, isRoot)
			}
		})
	}
}
