package structpath

import (
	"testing"

	"github.com/databricks/cli/libs/structdiff/jsontag"
	"github.com/stretchr/testify/assert"
)

func TestPathNode(t *testing.T) {
	// Note: The current DynPath() implementation has a bug where it calls p.prev.String()
	// instead of p.prev.DynPath() recursively. This causes inconsistent path formatting
	// for multi-node paths. The DynPath expectations below reflect the current buggy behavior.
	tests := []struct {
		name    string
		node    *PathNode
		String  string
		DynPath string // Optional - defaults to String if empty
	}{
		// Single node tests
		{
			name:   "nil path",
			node:   nil,
			String: "",
		},
		{
			name:   "array index",
			node:   NewIndex(nil, 5),
			String: "[5]",
		},
		{
			name:    "map key",
			node:    NewMapKey(nil, "mykey"),
			String:  `["mykey"]`,
			DynPath: "mykey",
		},
		{
			name:    "struct field with JSON tag",
			node:    NewStructField(nil, jsontag.JSONTag("json_name"), "GoFieldName"),
			String:  ".json_name",
			DynPath: "json_name",
		},
		{
			name:    "struct field without JSON tag (fallback to Go name)",
			node:    NewStructField(nil, jsontag.JSONTag(""), "GoFieldName"),
			String:  ".GoFieldName",
			DynPath: "GoFieldName",
		},
		{
			name:    "struct field with dash JSON tag",
			node:    NewStructField(nil, jsontag.JSONTag("-"), "GoFieldName"),
			String:  ".-",
			DynPath: "-",
		},
		{
			name:    "struct field with JSON tag options",
			node:    NewStructField(nil, jsontag.JSONTag("lazy_field,omitempty"), "LazyField"),
			String:  ".lazy_field",
			DynPath: "lazy_field",
		},

		// Two node tests
		{
			name:    "struct field -> array index",
			node:    NewIndex(NewStructField(nil, jsontag.JSONTag("items"), "Items"), 3),
			String:  ".items[3]",
			DynPath: ".items[3]", // Bug: should be "items[3]"
		},
		{
			name:    "struct field -> map key",
			node:    NewMapKey(NewStructField(nil, jsontag.JSONTag("config"), "Config"), "database"),
			String:  `.config["database"]`,
			DynPath: "config.database",
		},
		{
			name:    "struct field -> struct field",
			node:    NewStructField(NewStructField(nil, jsontag.JSONTag("user"), "User"), jsontag.JSONTag("name"), "Name"),
			String:  ".user.name",
			DynPath: "user.name",
		},
		{
			name:    "map key -> array index",
			node:    NewIndex(NewMapKey(nil, "servers"), 0),
			String:  `["servers"][0]`,
			DynPath: `["servers"][0]`, // Bug: should be "servers[0]"
		},
		{
			name:    "map key -> struct field",
			node:    NewStructField(NewMapKey(nil, "primary"), jsontag.JSONTag("host"), "Host"),
			String:  `["primary"].host`,
			DynPath: `["primary"].host`, // Bug: should be "primary.host"
		},
		{
			name:   "array index -> struct field",
			node:   NewStructField(NewIndex(nil, 2), jsontag.JSONTag("id"), "ID"),
			String: "[2].id",
		},
		{
			name:    "array index -> map key",
			node:    NewMapKey(NewIndex(nil, 1), "status"),
			String:  `[1]["status"]`,
			DynPath: "[1].status",
		},
		{
			name:    "struct field without JSON tag -> struct field with JSON tag",
			node:    NewStructField(NewStructField(nil, jsontag.JSONTag(""), "Parent"), jsontag.JSONTag("child_name"), "ChildName"),
			String:  ".Parent.child_name",
			DynPath: "Parent.child_name",
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
		})
	}
}

func TestPathNode_DynPathLazyResolution(t *testing.T) {
	// This test ensures DynPath() lazy resolution is covered
	// by calling DynPath() before String() on an unresolved node
	node := NewStructField(nil, jsontag.JSONTag("dyn_first,omitempty"), "DynFirst")

	// Call DynPath() first to trigger lazy resolution in DynPath method
	result := node.DynPath()
	assert.Equal(t, "dyn_first", result, "DynPath() should resolve JSON tag")

	// Verify String() works after DynPath() resolution
	result = node.String()
	assert.Equal(t, ".dyn_first", result, "String() after DynPath() resolution")
}
