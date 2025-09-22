package structpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathNode(t *testing.T) {
	tests := []struct {
		name        string
		node        *PathNode
		String      string
		Index       any
		MapKey      any
		Field       any
		Root        any
		DotStar     bool
		BracketStar bool
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
			name:   "map key",
			node:   NewMapKey(nil, "mykey"),
			String: `['mykey']`,
			MapKey: "mykey",
		},
		{
			name:   "struct field with JSON tag",
			node:   NewStructField(nil, "json_name"),
			String: "json_name",
			Field:  "json_name",
		},
		{
			name:   "struct field without JSON tag (fallback to Go name)",
			node:   NewStructField(nil, "GoFieldName"),
			String: "GoFieldName",
			Field:  "GoFieldName",
		},
		{
			name:   "struct field with dash JSON tag",
			node:   NewStructField(nil, "-"),
			String: "-",
			Field:  "-",
		},
		{
			name:   "struct field with JSON tag options",
			node:   NewStructField(nil, "lazy_field"),
			String: "lazy_field",
			Field:  "lazy_field",
		},
		{
			name:    "dot star",
			node:    NewDotStar(nil),
			String:  "*",
			DotStar: true,
		},
		{
			name:        "bracket star",
			node:        NewBracketStar(nil),
			String:      "[*]",
			BracketStar: true,
		},

		// Two node tests
		{
			name:   "struct field -> array index",
			node:   NewIndex(NewStructField(nil, "items"), 3),
			String: "items[3]",
			Index:  3,
		},
		{
			name:   "struct field -> map key",
			node:   NewMapKey(NewStructField(nil, "config"), "database"),
			String: `config['database']`,
			MapKey: "database",
		},
		{
			name:   "struct field -> struct field",
			node:   NewStructField(NewStructField(nil, "user"), "name"),
			String: "user.name",
			Field:  "name",
		},
		{
			name:   "map key -> array index",
			node:   NewIndex(NewMapKey(nil, "servers"), 0),
			String: `['servers'][0]`,
			Index:  0,
		},
		{
			name:   "map key -> struct field",
			node:   NewStructField(NewMapKey(nil, "primary"), "host"),
			String: `['primary'].host`,
			Field:  "host",
		},
		{
			name:   "array index -> struct field",
			node:   NewStructField(NewIndex(nil, 2), "id"),
			String: "[2].id",
			Field:  "id",
		},
		{
			name:   "array index -> map key",
			node:   NewMapKey(NewIndex(nil, 1), "status"),
			String: `[1]['status']`,
			MapKey: "status",
		},
		{
			name:   "struct field without JSON tag -> struct field with JSON tag",
			node:   NewStructField(NewStructField(nil, "Parent"), "child_name"),
			String: "Parent.child_name",
			Field:  "child_name",
		},
		{
			name:    "dot star with parent",
			node:    NewDotStar(NewStructField(nil, "Parent")),
			String:  "Parent.*",
			DotStar: true,
		},
		{
			name:        "bracket star with parent",
			node:        NewBracketStar(NewStructField(nil, "Parent")),
			String:      "Parent[*]",
			BracketStar: true,
		},

		// Edge cases with special characters in map keys
		{
			name:   "map key with single quote",
			node:   NewMapKey(nil, "key's"),
			String: `['key''s']`,
			MapKey: "key's",
		},
		{
			name:   "map key with multiple single quotes",
			node:   NewMapKey(nil, "''"),
			String: `['''''']`,
			MapKey: "''",
		},
		{
			name:   "empty map key",
			node:   NewMapKey(nil, ""),
			String: `['']`,
			MapKey: "",
		},
		{
			name: "complex path",
			node: NewStructField(
				NewIndex(
					NewMapKey(
						NewStructField(
							NewStructField(nil, "user"),
							"settings"),
						"theme"),
					0),
				"color"),
			String: "user.settings['theme'][0].color",
			Field:  "color",
		},
		{
			name:   "field with special characters",
			node:   NewStructField(nil, "field@name:with#symbols!"),
			String: "field@name:with#symbols!",
			Field:  "field@name:with#symbols!",
		},
		{
			name:   "field with spaces",
			node:   NewStringKey(nil, "field with spaces"),
			String: "['field with spaces']",
			MapKey: "field with spaces",
		},
		{
			name:   "field starting with digit",
			node:   NewStructField(nil, "123field"),
			String: "123field",
			Field:  "123field",
		},
		{
			name:   "field with unicode",
			node:   NewStructField(nil, "名前🙂"),
			String: "名前🙂",
			Field:  "名前🙂",
		},
		{
			name:   "map key with reserved characters",
			node:   NewMapKey(nil, "key\x00[],`"),
			String: "['key\x00[],`']",
			MapKey: "key\x00[],`",
		},

		// Additional dot-star pattern tests
		{
			name:    "field dot star",
			node:    NewDotStar(NewStructField(nil, "bla")),
			String:  "bla.*",
			DotStar: true,
		},
		{
			name:   "field dot star dot field",
			node:   NewStructField(NewDotStar(NewStructField(nil, "bla")), "foo"),
			String: "bla.*.foo",
			Field:  "foo",
		},
		{
			name:   "field dot star bracket index",
			node:   NewIndex(NewDotStar(NewStructField(nil, "bla")), 0),
			String: "bla.*[0]",
			Index:  0,
		},
		{
			name:        "field dot star bracket star",
			node:        NewBracketStar(NewDotStar(NewStructField(nil, "bla"))),
			String:      "bla.*[*]",
			BracketStar: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test String() method
			result := tt.node.String()
			assert.Equal(t, tt.String, result, "String() method")

			// Test roundtrip conversion: String() -> Parse() -> String()
			parsed, err := Parse(tt.String)
			if assert.NoError(t, err, "Parse() should not error") {
				assert.Equal(t, tt.node, parsed)
				roundtripResult := parsed.String()
				assert.Equal(t, tt.String, roundtripResult, "Roundtrip conversion should be identical")
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

			// DotStar and BracketStar
			assert.Equal(t, tt.DotStar, tt.node.DotStar())
			assert.Equal(t, tt.BracketStar, tt.node.BracketStar())
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
			error: "unexpected end of input after '.'",
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
			error: "unexpected end of input after '['",
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
			error: "unexpected end of input while parsing map key",
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
			error: "unexpected end of input while parsing map key",
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
			error: "unexpected end of input while parsing map key",
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

		// Invalid dot-star patterns
		{
			name:  "dot star followed by field name",
			input: "bla.*foo",
			error: "unexpected character 'f' after '.*' at position 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if assert.Error(t, err) {
				assert.Equal(t, tt.error, err.Error())
			}
		})
	}
}

func TestNewIndexPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, r.(string), "index must be non-negative")
		}
	}()
	NewIndex(nil, -1) // Should panic
	t.Error("Expected panic did not occur")
}
