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
		StringKey   any
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
			name:      "map key",
			node:      NewStringKey(nil, "mykey"),
			String:    `mykey`,
			StringKey: "mykey",
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
			node:   NewIndex(NewStringKey(nil, "items"), 3),
			String: "items[3]",
			Index:  3,
		},
		{
			name:      "struct field -> map key",
			node:      NewStringKey(NewStringKey(nil, "config"), "database.name"),
			String:    `config['database.name']`,
			StringKey: "database.name",
		},
		{
			name:      "struct field -> struct field",
			node:      NewStringKey(NewStringKey(nil, "user"), "name"),
			String:    "user.name",
			StringKey: "name",
		},
		{
			name:   "map key -> array index",
			node:   NewIndex(NewStringKey(nil, "servers list"), 0),
			String: `['servers list'][0]`,
			Index:  0,
		},
		{
			name:      "array index -> struct field",
			node:      NewStringKey(NewIndex(nil, 2), "id"),
			String:    "[2].id",
			StringKey: "id",
		},
		{
			name:      "array index -> map key",
			node:      NewStringKey(NewIndex(nil, 1), "status{}"),
			String:    `[1]['status{}']`,
			StringKey: "status{}",
		},
		{
			name:    "dot star with parent",
			node:    NewDotStar(NewStringKey(nil, "Parent")),
			String:  "Parent.*",
			DotStar: true,
		},
		{
			name:        "bracket star with parent",
			node:        NewBracketStar(NewStringKey(nil, "Parent")),
			String:      "Parent[*]",
			BracketStar: true,
		},

		// Edge cases with special characters in map keys
		{
			name:      "map key with single quote",
			node:      NewStringKey(nil, "key's"),
			String:    `['key''s']`,
			StringKey: "key's",
		},
		{
			name:      "map key with multiple single quotes",
			node:      NewStringKey(nil, "''"),
			String:    `['''''']`,
			StringKey: "''",
		},
		{
			name:      "empty map key",
			node:      NewStringKey(nil, ""),
			String:    `['']`,
			StringKey: "",
		},
		{
			name: "complex path",
			node: NewStringKey(
				NewIndex(
					NewStringKey(
						NewStringKey(
							NewStringKey(nil, "user"),
							"settings"),
						"theme.list"),
					0),
				"color"),
			String:    "user.settings['theme.list'][0].color",
			StringKey: "color",
		},
		{
			name:      "field with special characters",
			node:      NewStringKey(nil, "field@name:with#symbols!"),
			String:    "field@name:with#symbols!",
			StringKey: "field@name:with#symbols!",
		},
		{
			name:      "field with spaces",
			node:      NewStringKey(nil, "field with spaces"),
			String:    "['field with spaces']",
			StringKey: "field with spaces",
		},
		{
			name:      "field starting with digit",
			node:      NewStringKey(nil, "123field"),
			String:    "123field",
			StringKey: "123field",
		},
		{
			name:      "field with unicode",
			node:      NewStringKey(nil, "名前🙂"),
			String:    "名前🙂",
			StringKey: "名前🙂",
		},
		{
			name:      "map key with reserved characters",
			node:      NewStringKey(nil, "key\x00[],`"),
			String:    "['key\x00[],`']",
			StringKey: "key\x00[],`",
		},

		{
			name:   "field dot star bracket index",
			node:   NewIndex(NewDotStar(NewStringKey(nil, "bla")), 0),
			String: "bla.*[0]",
			Index:  0,
		},
		{
			name:        "field dot star bracket star",
			node:        NewBracketStar(NewDotStar(NewStringKey(nil, "bla"))),
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

			gotStringKey, isStringKey := tt.node.StringKey()
			if tt.StringKey == nil {
				assert.Equal(t, "", gotStringKey)
				assert.False(t, isStringKey)
			} else {
				expected := tt.StringKey.(string)
				assert.Equal(t, expected, gotStringKey)
				assert.True(t, isStringKey)
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

func TestPrefixAndSkipPrefix(t *testing.T) {
	tests := []struct {
		input      string
		n          int
		prefix     string
		skipPrefix string
	}{
		{
			input:      "resources.jobs.my_job.tasks[0].name",
			n:          0,
			prefix:     "",
			skipPrefix: "resources.jobs.my_job.tasks[0].name",
		},
		{
			input:      "resources.jobs.my_job.tasks[0].name",
			n:          1,
			prefix:     "resources",
			skipPrefix: "jobs.my_job.tasks[0].name",
		},
		{
			input:      "resources.jobs.my_job.tasks[0].name",
			n:          3,
			prefix:     "resources.jobs.my_job",
			skipPrefix: "tasks[0].name",
		},
		{
			input:      "resources.jobs.my_job.tasks[0].name",
			n:          5,
			prefix:     "resources.jobs.my_job.tasks[0]",
			skipPrefix: "name",
		},
		{
			input:      "resources.jobs.my_job.tasks[0].name",
			n:          6,
			prefix:     "resources.jobs.my_job.tasks[0].name",
			skipPrefix: "",
		},
		{
			input:      "resources.jobs.my_job.tasks[0].name",
			n:          10,
			prefix:     "resources.jobs.my_job.tasks[0].name",
			skipPrefix: "",
		},
		{
			input:      "resources.jobs.my_job.tasks[0].name",
			n:          -1,
			prefix:     "",
			skipPrefix: "resources.jobs.my_job.tasks[0].name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			path, err := Parse(tt.input)
			assert.NoError(t, err)

			// Test Prefix
			prefixResult := path.Prefix(tt.n)
			if tt.prefix == "" {
				assert.Nil(t, prefixResult)
			} else {
				assert.NotNil(t, prefixResult)
				assert.Equal(t, tt.prefix, prefixResult.String())
			}

			// Test SkipPrefix
			skipResult := path.SkipPrefix(tt.n)
			if tt.skipPrefix == "" {
				assert.Nil(t, skipResult)
			} else {
				assert.NotNil(t, skipResult)
				assert.Equal(t, tt.skipPrefix, skipResult.String())
			}
		})
	}
}

func TestLen(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{
			input:    "",
			expected: 0,
		},
		{
			input:    "field",
			expected: 1,
		},
		{
			input:    "resources.jobs['my_job'].tasks[0]",
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var path *PathNode
			var err error
			path, err = Parse(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, path.Len())
		})
	}
}
