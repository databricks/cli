package structpath

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathNode(t *testing.T) {
	tests := []struct {
		name          string
		node          *PathNode
		String        string
		DynPath       string // Only set when different from String
		IgnoreDynPath bool   // Do not test DynPath
		Index         any
		MapKey        any
		Field         any
		Root          any
		AnyKey        bool
		AnyIndex      bool
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
			String:  `['mykey']`,
			DynPath: "mykey",
			MapKey:  "mykey",
		},
		{
			name:   "struct field with JSON tag",
			node:   NewStructField(nil, reflect.StructTag(`json:"json_name"`), "GoFieldName"),
			String: "json_name",
			Field:  "json_name",
		},
		{
			name:   "struct field without JSON tag (fallback to Go name)",
			node:   NewStructField(nil, reflect.StructTag(""), "GoFieldName"),
			String: "GoFieldName",
			Field:  "GoFieldName",
		},
		{
			name:   "struct field with dash JSON tag",
			node:   NewStructField(nil, reflect.StructTag(`json:"-"`), "GoFieldName"),
			String: "-",
			Field:  "-",
		},
		{
			name:   "struct field with JSON tag options",
			node:   NewStructField(nil, reflect.StructTag(`json:"lazy_field,omitempty"`), "LazyField"),
			String: "lazy_field",
			Field:  "lazy_field",
		},
		{
			name:    "any key",
			node:    NewAnyKey(nil),
			String:  "[*]",
			DynPath: "*",
			AnyKey:  true,
		},
		{
			name:     "any index",
			node:     NewAnyIndex(nil),
			String:   "[*]",
			AnyIndex: true,
		},

		// Two node tests
		{
			name:   "struct field -> array index",
			node:   NewIndex(NewStructField(nil, reflect.StructTag(`json:"items"`), "Items"), 3),
			String: "items[3]",
			Index:  3,
		},
		{
			name:    "struct field -> map key",
			node:    NewMapKey(NewStructField(nil, reflect.StructTag(`json:"config"`), "Config"), "database"),
			String:  `config['database']`,
			DynPath: "config.database",
			MapKey:  "database",
		},
		{
			name:   "struct field -> struct field",
			node:   NewStructField(NewStructField(nil, reflect.StructTag(`json:"user"`), "User"), reflect.StructTag(`json:"name"`), "Name"),
			String: "user.name",
			Field:  "name",
		},
		{
			name:    "map key -> array index",
			node:    NewIndex(NewMapKey(nil, "servers"), 0),
			String:  `['servers'][0]`,
			DynPath: "servers[0]",
			Index:   0,
		},
		{
			name:    "map key -> struct field",
			node:    NewStructField(NewMapKey(nil, "primary"), reflect.StructTag(`json:"host"`), "Host"),
			String:  `['primary'].host`,
			DynPath: `primary.host`,
			Field:   "host",
		},
		{
			name:   "array index -> struct field",
			node:   NewStructField(NewIndex(nil, 2), reflect.StructTag(`json:"id"`), "ID"),
			String: "[2].id",
			Field:  "id",
		},
		{
			name:    "array index -> map key",
			node:    NewMapKey(NewIndex(nil, 1), "status"),
			String:  `[1]['status']`,
			DynPath: "[1].status",
			MapKey:  "status",
		},
		{
			name:   "struct field without JSON tag -> struct field with JSON tag",
			node:   NewStructField(NewStructField(nil, reflect.StructTag(""), "Parent"), reflect.StructTag(`json:"child_name"`), "ChildName"),
			String: "Parent.child_name",
			Field:  "child_name",
		},
		{
			name:    "any key",
			node:    NewAnyKey(NewStructField(nil, reflect.StructTag(""), "Parent")),
			String:  "Parent[*]",
			DynPath: "Parent.*",
			AnyKey:  true,
		},
		{
			name:     "any index",
			node:     NewAnyIndex(NewStructField(nil, reflect.StructTag(""), "Parent")),
			String:   "Parent[*]",
			AnyIndex: true,
		},
		// Edge cases with special characters in map keys
		{
			name:    "map key with single quote",
			node:    NewMapKey(nil, "key's"),
			String:  `['key''s']`,
			DynPath: "key's",
			MapKey:  "key's",
		},
		{
			name:    "map key with multiple single quotes",
			node:    NewMapKey(nil, "''"),
			String:  `['''''']`,
			DynPath: "''",
			MapKey:  "''",
		},
		{
			name:          "empty map key",
			node:          NewMapKey(nil, ""),
			String:        `['']`,
			IgnoreDynPath: true,
			MapKey:        "",
		},
		{
			name: "complex path",
			node: NewStructField(
				NewIndex(
					NewMapKey(
						NewStructField(
							NewStructField(nil, reflect.StructTag(`json:"user"`), "User"),
							reflect.StructTag(`json:"settings"`), "Settings"),
						"theme"),
					0),
				reflect.StructTag(`json:"color"`), "Color"),
			String:  "user.settings['theme'][0].color",
			DynPath: "user.settings.theme[0].color",
			Field:   "color",
		},
		{
			name:   "field with special characters",
			node:   NewStructField(nil, reflect.StructTag(""), "field@name:with#symbols!"),
			String: "field@name:with#symbols!",
			Field:  "field@name:with#symbols!",
		},
		{
			name:   "field with spaces",
			node:   NewStructField(nil, reflect.StructTag(""), "field with spaces"),
			String: "field with spaces",
			Field:  "field with spaces",
		},
		{
			name:   "field starting with digit",
			node:   NewStructField(nil, reflect.StructTag(""), "123field"),
			String: "123field",
			Field:  "123field",
		},
		{
			name:   "field with unicode",
			node:   NewStructField(nil, reflect.StructTag(""), "名前🙂"),
			String: "名前🙂",
			Field:  "名前🙂",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test String() method
			result := tt.node.String()
			assert.Equal(t, tt.String, result, "String() method")

			// Test roundtrip conversion: String() -> Parse() -> String()
			parsed, err := Parse(tt.String)
			assert.NoError(t, err, "Parse() should not error")
			if parsed != nil {
				roundtripResult := parsed.String()
				assert.Equal(t, tt.String, roundtripResult, "Roundtrip conversion should be identical")
			}

			if !tt.IgnoreDynPath {
				dynResult := tt.node.DynPath()
				expectedDyn := tt.String
				if tt.DynPath != "" {
					expectedDyn = tt.DynPath
					// Enforce rule: DynPath should only be set when different from String
					assert.NotEqual(t, expectedDyn, tt.String, "Test case %q: DynPath should only be set when different from String", tt.name)
				}
				assert.Equal(t, expectedDyn, dynResult, "DynPath() method")
			}

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

			// IsRoot
			isRoot := tt.node.IsRoot()
			if tt.Root == nil {
				assert.False(t, isRoot)
			} else {
				assert.True(t, isRoot)
			}

			// AnyKey, AnyIndex
			assert.Equal(t, tt.AnyKey, tt.node.AnyKey())
			assert.Equal(t, tt.AnyIndex, tt.node.AnyIndex())
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		error string
	}{
		{
			name:  "starts with dot",
			input: ".field",
			error: "unexpected character '.' at position 0",
		},
		{
			name:  "ends with dot",
			input: "field.",
			error: "expected field name after '.' but reached end of input",
		},
		{
			name:  "consecutive dots",
			input: "field..child",
			error: "expected field name after '.' but got '.' at position 6",
		},
		{
			name:  "dot before bracket",
			input: "field.[0]",
			error: "expected field name after '.' but got '[' at position 6",
		},
		{
			name:  "unclosed bracket",
			input: "field[",
			error: "unexpected end of input: unclosed bracket",
		},
		{
			name:  "empty brackets",
			input: "field[]",
			error: "unexpected character ']' after '[' at position 6",
		},
		{
			name:  "invalid array index",
			input: "[abc]",
			error: "unexpected character 'a' after '[' at position 1",
		},
		{
			name:  "negative array index",
			input: "[-1]",
			error: "unexpected character '-' after '[' at position 1",
		},
		{
			name:  "unclosed map key quote",
			input: "['key]",
			error: "unterminated map key at position 5",
		},
		{
			name:  "empty field name",
			input: ".field",
			error: "unexpected character '.' at position 0",
		},
		{
			name:  "field with comma",
			input: "field,name",
			error: "invalid character ',' in field name at position 5",
		},
		{
			name:  "field with double quote",
			input: "field\"name",
			error: "invalid character '\"' in field name at position 5",
		},
		{
			name:  "field with backtick",
			input: "field`name",
			error: "invalid character '`' in field name at position 5",
		},
		{
			name:  "index overflow",
			input: "[99999999999999999999999999999999999999]",
			error: "invalid index '99999999999999999999999999999999999999' at position 1",
		},
		{
			name:  "parser error - invalid state",
			input: "field[']",
			error: "unterminated map key at position 7",
		},
		{
			name:  "map key with null byte",
			input: "['key\x00']",
			error: "unterminated map key at position 5",
		},
		{
			name:  "field with right bracket",
			input: "field]name",
			error: "invalid character ']' in field name at position 5",
		},
		{
			name:  "map key quote escape at end",
			input: "['key'''",
			error: "unexpected end of input after quote in map key",
		},
		{
			name:  "unexpected character in map key quote state",
			input: "['key'x]",
			error: "unexpected character 'x' after quote in map key at position 6",
		},
		{
			name:  "wildcard with invalid character",
			input: "[*x]",
			error: "unexpected character 'x' after '*' at position 2",
		},
		{
			name:  "invalid character in bracket",
			input: "field[name",
			error: "unexpected character 'n' after '[' at position 6",
		},
		{
			name:  "unexpected character after valid path",
			input: "field[0]x",
			error: "unexpected character 'x' at position 8",
		},
		{
			name:  "map key terminated by EOF",
			input: "['key",
			error: "unterminated map key at position 5",
		},
		{
			name:  "starts with comma",
			input: ",field",
			error: "unexpected character ',' at position 0",
		},
		{
			name:  "starts with right bracket",
			input: "]field",
			error: "unexpected character ']' at position 0",
		},
		{
			name:  "starts with backtick",
			input: "`field",
			error: "unexpected character '`' at position 0",
		},
		{
			name:  "incomplete index - missing closing bracket",
			input: "field[123",
			error: "unexpected end of input while parsing index",
		},
		{
			name:  "incomplete wildcard",
			input: "field[*",
			error: "unexpected end of input after wildcard '*'",
		},
		{
			name:  "incomplete map key quote",
			input: "field['key'",
			error: "unexpected end of input after quote in map key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			assert.Error(t, err)
			assert.Equal(t, tt.error, err.Error())
		})
	}
}

func TestNewIndexPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, r.(string), "index msut be non-negative")
		}
	}()
	NewIndex(nil, -1) // Should panic
	t.Error("Expected panic did not occur")
}
