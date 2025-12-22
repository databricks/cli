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
		KeyValue    []string // [key, value] or nil
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
			node:      NewDotString(nil, "mykey"),
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
		{
			name:     "key value",
			node:     NewKeyValue(nil, "name", "foo"),
			String:   "[name='foo']",
			KeyValue: []string{"name", "foo"},
		},

		// Two node tests
		{
			name:   "struct field -> array index",
			node:   NewIndex(NewDotString(nil, "items"), 3),
			String: "items[3]",
			Index:  3,
		},
		{
			name:      "struct field -> map key",
			node:      NewBracketString(NewDotString(nil, "config"), "database.name"),
			String:    `config['database.name']`,
			StringKey: "database.name",
		},
		{
			name:      "struct field -> struct field",
			node:      NewDotString(NewDotString(nil, "user"), "name"),
			String:    "user.name",
			StringKey: "name",
		},
		{
			name:   "map key -> array index",
			node:   NewIndex(NewBracketString(nil, "servers list"), 0),
			String: `['servers list'][0]`,
			Index:  0,
		},
		{
			name:      "array index -> struct field",
			node:      NewDotString(NewIndex(nil, 2), "id"),
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

		// Key-value tests
		{
			name:     "key value with parent",
			node:     NewKeyValue(NewStringKey(nil, "tasks"), "task_key", "my_task"),
			String:   "tasks[task_key='my_task']",
			KeyValue: []string{"task_key", "my_task"},
		},
		{
			name:      "key value then field",
			node:      NewStringKey(NewKeyValue(nil, "name", "foo"), "id"),
			String:    "[name='foo'].id",
			StringKey: "id",
		},
		{
			name:     "key value with quote in value",
			node:     NewKeyValue(nil, "name", "it's"),
			String:   "[name='it''s']",
			KeyValue: []string{"name", "it's"},
		},
		{
			name:     "key value with empty value",
			node:     NewKeyValue(nil, "key", ""),
			String:   "[key='']",
			KeyValue: []string{"key", ""},
		},
		{
			name:      "complex path with key value",
			node:      NewStringKey(NewKeyValue(NewStringKey(NewStringKey(nil, "resources"), "jobs"), "task_key", "my_task"), "notebook_task"),
			String:    "resources.jobs[task_key='my_task'].notebook_task",
			StringKey: "notebook_task",
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

			// KeyValue
			gotKey, gotValue, isKeyValue := tt.node.KeyValue()
			if tt.KeyValue == nil {
				assert.Equal(t, "", gotKey)
				assert.Equal(t, "", gotValue)
				assert.False(t, isKeyValue)
			} else {
				assert.Equal(t, tt.KeyValue[0], gotKey)
				assert.Equal(t, tt.KeyValue[1], gotValue)
				assert.True(t, isKeyValue)
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
			error: "unexpected character ']' in key-value key at position 4",
		},
		{
			name:  "negative array index",
			input: "[-1]",
			error: "unexpected character ']' in key-value key at position 3",
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
			error: "unexpected end of input while parsing key-value key",
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

		// Invalid key-value patterns
		{
			name:  "key-value missing equals",
			input: "[name'value']",
			error: "unexpected character ''' in key-value key at position 5",
		},
		{
			name:  "key-value missing value quote",
			input: "[name=value]",
			error: "expected quote after '=' but got 'v' at position 6",
		},
		{
			name:  "key-value incomplete key",
			input: "[name",
			error: "unexpected end of input while parsing key-value key",
		},
		{
			name:  "key-value incomplete after equals",
			input: "[name=",
			error: "unexpected end of input after '=' in key-value",
		},
		{
			name:  "key-value incomplete value",
			input: "[name='value",
			error: "unexpected end of input while parsing key-value value",
		},
		{
			name:  "key-value incomplete after value quote",
			input: "[name='value'",
			error: "unexpected end of input after quote in key-value value",
		},
		{
			name:  "key-value invalid char after value quote",
			input: "[name='value'x]",
			error: "unexpected character 'x' after quote in key-value at position 13",
		},
		{
			name:  "double quotes are not supported a.t.m",
			input: "[name=\"value\"]",
			error: "expected quote after '=' but got '\"' at position 6",
		},
		{
			name:  "mixed quotes never going to be supported",
			input: "[name='value\"]",
			error: "unexpected end of input while parsing key-value value",
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

func TestPureReferenceToPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{
			name:     "simple reference",
			input:    "${resources.jobs.foo.id}",
			expected: "resources.jobs.foo.id",
			ok:       true,
		},
		{
			name:     "simple reference",
			input:    "${resources.jobs.foo.tasks[1].env.key}",
			expected: "resources.jobs.foo.tasks[1].env.key",
			ok:       true,
		},
		{
			name:  "complex nested reference",
			input: "${var.resources.jobs['my_job'].tasks[0]}",
			// we use regex from dyn module which only support integers inside brackets:
			// expected: "resources.jobs['my_job'].tasks[0]",
		},
		{
			name:  "not a pure reference",
			input: "prefix_${var.field}",
		},
		{
			name:  "not a variable reference",
			input: "plain_string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathNode, ok := PureReferenceToPath(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.NotNil(t, pathNode)
				assert.Equal(t, tt.expected, pathNode.String())
			} else {
				assert.Nil(t, pathNode)
			}
		})
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefix   string
		expected bool
	}{
		// Edge cases
		{
			name:     "empty prefix",
			s:        "a.b.c",
			prefix:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			prefix:   "a",
			expected: false,
		},
		{
			name:     "exact match",
			s:        "config",
			prefix:   "config",
			expected: true,
		},

		// Correct matches - path boundary aware
		{
			name:     "simple field match",
			s:        "a.b",
			prefix:   "a",
			expected: true,
		},
		{
			name:     "nested field match",
			s:        "config.database.name",
			prefix:   "config.database",
			expected: true,
		},
		{
			name:     "field with array index",
			s:        "items[3].name",
			prefix:   "items",
			expected: true,
		},
		{
			name:     "array with prefix match",
			s:        "items[0].name",
			prefix:   "items[0]",
			expected: true,
		},
		{
			name:     "field with bracket notation",
			s:        "config['spark.conf'].value",
			prefix:   "config['spark.conf']",
			expected: true,
		},

		// Incorrect matches - should NOT match
		{
			name:     "substring match without boundary",
			s:        "ai_gateway",
			prefix:   "ai",
			expected: false,
		},
		{
			name:     "different nested field",
			s:        "configuration.name",
			prefix:   "config",
			expected: false,
		},

		// wildcard patterns are NOT supported - treated as literals
		{
			name:     "regex pattern not respected - star quantifier",
			s:        "aaa",
			prefix:   "a*",
			expected: false,
		},
		{
			name:     "regex pattern not respected - bracket class",
			s:        "a[1]",
			prefix:   "a[*]",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPrefix(tt.s, tt.prefix)
			assert.Equal(t, tt.expected, result, "HasPrefix(%q, %q)", tt.s, tt.prefix)
		})
	}
}
