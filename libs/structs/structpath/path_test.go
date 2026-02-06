package structpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestPathAndPatternNode(t *testing.T) {
	tests := []struct {
		name        string
		pathNode    *PathNode    // nil for wildcard-only patterns
		patternNode *PatternNode // always set
		String      string
		Index       any
		StringKey   any
		KeyValue    []string // [key, value] or nil
		Root        any
		DotStar     bool
		BracketStar bool
		PathError   string // expected error when parsing as path (for wildcards)
	}{
		// Single node tests
		{
			name:        "nil path",
			pathNode:    nil,
			patternNode: nil,
			String:      "",
			Root:        true,
		},
		{
			name:        "array index",
			pathNode:    NewIndex(nil, 5),
			patternNode: NewPatternIndex(nil, 5),
			String:      "[5]",
			Index:       5,
		},
		{
			name:        "map key",
			pathNode:    NewDotString(nil, "mykey"),
			patternNode: NewPatternStringKey(nil, "mykey"),
			String:      `mykey`,
			StringKey:   "mykey",
		},
		{
			name:        "key value",
			pathNode:    NewKeyValue(nil, "name", "foo"),
			patternNode: NewPatternKeyValue(nil, "name", "foo"),
			String:      "[name='foo']",
			KeyValue:    []string{"name", "foo"},
		},

		// Two node tests
		{
			name:        "struct field -> array index",
			pathNode:    NewIndex(NewDotString(nil, "items"), 3),
			patternNode: NewPatternIndex(NewPatternStringKey(nil, "items"), 3),
			String:      "items[3]",
			Index:       3,
		},
		{
			name:        "struct field -> map key",
			pathNode:    NewBracketString(NewDotString(nil, "config"), "database.name"),
			patternNode: NewPatternBracketString(NewPatternStringKey(nil, "config"), "database.name"),
			String:      `config['database.name']`,
			StringKey:   "database.name",
		},
		{
			name:        "struct field -> struct field",
			pathNode:    NewDotString(NewDotString(nil, "user"), "name"),
			patternNode: NewPatternDotString(NewPatternStringKey(nil, "user"), "name"),
			String:      "user.name",
			StringKey:   "name",
		},
		{
			name:        "map key -> array index",
			pathNode:    NewIndex(NewBracketString(nil, "servers list"), 0),
			patternNode: NewPatternIndex(NewPatternBracketString(nil, "servers list"), 0),
			String:      `['servers list'][0]`,
			Index:       0,
		},
		{
			name:        "array index -> struct field",
			pathNode:    NewDotString(NewIndex(nil, 2), "id"),
			patternNode: NewPatternDotString(NewPatternIndex(nil, 2), "id"),
			String:      "[2].id",
			StringKey:   "id",
		},
		{
			name:        "array index -> map key",
			pathNode:    NewStringKey(NewIndex(nil, 1), "status{}"),
			patternNode: NewPatternStringKey(NewPatternIndex(nil, 1), "status{}"),
			String:      `[1]['status{}']`,
			StringKey:   "status{}",
		},

		// Edge cases with special characters in map keys
		{
			name:        "map key with single quote",
			pathNode:    NewStringKey(nil, "key's"),
			patternNode: NewPatternStringKey(nil, "key's"),
			String:      `['key''s']`,
			StringKey:   "key's",
		},
		{
			name:        "map key with multiple single quotes",
			pathNode:    NewStringKey(nil, "''"),
			patternNode: NewPatternStringKey(nil, "''"),
			String:      `['''''']`,
			StringKey:   "''",
		},
		{
			name:        "empty map key",
			pathNode:    NewStringKey(nil, ""),
			patternNode: NewPatternStringKey(nil, ""),
			String:      `['']`,
			StringKey:   "",
		},
		{
			name: "complex path",
			pathNode: NewStringKey(
				NewIndex(
					NewStringKey(
						NewStringKey(
							NewStringKey(nil, "user"),
							"settings"),
						"theme.list"),
					0),
				"color"),
			patternNode: NewPatternStringKey(
				NewPatternIndex(
					NewPatternStringKey(
						NewPatternStringKey(
							NewPatternStringKey(nil, "user"),
							"settings"),
						"theme.list"),
					0),
				"color"),
			String:    "user.settings['theme.list'][0].color",
			StringKey: "color",
		},
		{
			name:        "field with special characters",
			pathNode:    NewStringKey(nil, "field@name:with#symbols!"),
			patternNode: NewPatternStringKey(nil, "field@name:with#symbols!"),
			String:      "field@name:with#symbols!",
			StringKey:   "field@name:with#symbols!",
		},
		{
			name:        "field with spaces",
			pathNode:    NewStringKey(nil, "field with spaces"),
			patternNode: NewPatternStringKey(nil, "field with spaces"),
			String:      "['field with spaces']",
			StringKey:   "field with spaces",
		},
		{
			name:        "field starting with digit",
			pathNode:    NewStringKey(nil, "123field"),
			patternNode: NewPatternStringKey(nil, "123field"),
			String:      "123field",
			StringKey:   "123field",
		},
		{
			name:        "field with unicode",
			pathNode:    NewStringKey(nil, "名前🙂"),
			patternNode: NewPatternStringKey(nil, "名前🙂"),
			String:      "名前🙂",
			StringKey:   "名前🙂",
		},
		{
			name:        "map key with reserved characters",
			pathNode:    NewStringKey(nil, "key\x00[],`"),
			patternNode: NewPatternStringKey(nil, "key\x00[],`"),
			String:      "['key\x00[],`']",
			StringKey:   "key\x00[],`",
		},

		// Key-value tests
		{
			name:        "key value with parent",
			pathNode:    NewKeyValue(NewStringKey(nil, "tasks"), "task_key", "my_task"),
			patternNode: NewPatternKeyValue(NewPatternStringKey(nil, "tasks"), "task_key", "my_task"),
			String:      "tasks[task_key='my_task']",
			KeyValue:    []string{"task_key", "my_task"},
		},
		{
			name:        "key value then field",
			pathNode:    NewStringKey(NewKeyValue(nil, "name", "foo"), "id"),
			patternNode: NewPatternStringKey(NewPatternKeyValue(nil, "name", "foo"), "id"),
			String:      "[name='foo'].id",
			StringKey:   "id",
		},
		{
			name:        "key value with quote in value",
			pathNode:    NewKeyValue(nil, "name", "it's"),
			patternNode: NewPatternKeyValue(nil, "name", "it's"),
			String:      "[name='it''s']",
			KeyValue:    []string{"name", "it's"},
		},
		{
			name:        "key value with empty value",
			pathNode:    NewKeyValue(nil, "key", ""),
			patternNode: NewPatternKeyValue(nil, "key", ""),
			String:      "[key='']",
			KeyValue:    []string{"key", ""},
		},
		{
			name:        "complex path with key value",
			pathNode:    NewStringKey(NewKeyValue(NewStringKey(NewStringKey(nil, "resources"), "jobs"), "task_key", "my_task"), "notebook_task"),
			patternNode: NewPatternStringKey(NewPatternKeyValue(NewPatternStringKey(NewPatternStringKey(nil, "resources"), "jobs"), "task_key", "my_task"), "notebook_task"),
			String:      "resources.jobs[task_key='my_task'].notebook_task",
			StringKey:   "notebook_task",
		},

		// Wildcard patterns (cannot be parsed as PathNode)
		{
			name:        "dot star",
			patternNode: NewPatternDotStar(nil),
			String:      "*",
			DotStar:     true,
			PathError:   "wildcards not allowed in path",
		},
		{
			name:        "bracket star",
			patternNode: NewPatternBracketStar(nil),
			String:      "[*]",
			BracketStar: true,
			PathError:   "wildcards not allowed in path",
		},
		{
			name:        "dot star with parent",
			patternNode: NewPatternDotStar(NewPatternStringKey(nil, "Parent")),
			String:      "Parent.*",
			DotStar:     true,
			PathError:   "wildcards not allowed in path",
		},
		{
			name:        "bracket star with parent",
			patternNode: NewPatternBracketStar(NewPatternStringKey(nil, "Parent")),
			String:      "Parent[*]",
			BracketStar: true,
			PathError:   "wildcards not allowed in path",
		},
		{
			name:        "field dot star bracket index",
			patternNode: NewPatternIndex(NewPatternDotStar(NewPatternStringKey(nil, "bla")), 0),
			String:      "bla.*[0]",
			PathError:   "wildcards not allowed in path",
		},
		{
			name:        "field dot star bracket star",
			patternNode: NewPatternBracketStar(NewPatternDotStar(NewPatternStringKey(nil, "bla"))),
			String:      "bla.*[*]",
			BracketStar: true,
			PathError:   "wildcards not allowed in path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pattern parsing and roundtrip
			parsedPattern, err := ParsePattern(tt.String)
			if assert.NoError(t, err, "ParsePattern() should not error") {
				assert.Equal(t, tt.patternNode, parsedPattern)
				assert.Equal(t, tt.String, parsedPattern.String(), "Pattern roundtrip")
			}

			// Test DotStar and BracketStar on pattern
			if tt.patternNode != nil {
				assert.Equal(t, tt.DotStar, tt.patternNode.DotStar())
				assert.Equal(t, tt.BracketStar, tt.patternNode.BracketStar())
			}

			// Test path parsing
			if tt.PathError != "" {
				// Wildcard pattern - should fail to parse as path
				_, err := ParsePath(tt.String)
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.PathError)
				}
			} else {
				// Concrete path - should parse successfully as both path and pattern
				parsedPath, err := ParsePath(tt.String)
				if assert.NoError(t, err, "ParsePath() should not error") {
					assert.Equal(t, tt.pathNode, parsedPath)
					assert.Equal(t, tt.String, parsedPath.String(), "Path roundtrip")
				}

				// Test PathNode-specific methods
				gotIndex, isIndex := tt.pathNode.Index()
				if tt.Index == nil {
					assert.Equal(t, -1, gotIndex)
					assert.False(t, isIndex)
				} else {
					expectedIndex := tt.Index.(int)
					assert.Equal(t, expectedIndex, gotIndex)
					assert.True(t, isIndex)
				}

				gotStringKey, isStringKey := tt.pathNode.StringKey()
				if tt.StringKey == nil {
					assert.Equal(t, "", gotStringKey)
					assert.False(t, isStringKey)
				} else {
					expected := tt.StringKey.(string)
					assert.Equal(t, expected, gotStringKey)
					assert.True(t, isStringKey)
				}

				// KeyValue
				gotKey, gotValue, isKeyValue := tt.pathNode.KeyValue()
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
				isRoot := tt.pathNode.IsRoot()
				if tt.Root == nil {
					assert.False(t, isRoot)
				} else {
					assert.True(t, isRoot)
				}
			}
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
			_, err := ParsePattern(tt.input) // Allow wildcards in error tests
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
		// Regression test for SkipPrefix bug: value field was missing in KeyValue nodes
		// Test all possible n values for completeness
		{
			input:      "resources.jobs[task_key='my_task'].notebook_task",
			n:          0,
			prefix:     "",
			skipPrefix: "resources.jobs[task_key='my_task'].notebook_task",
		},
		{
			input:      "resources.jobs[task_key='my_task'].notebook_task",
			n:          1,
			prefix:     "resources",
			skipPrefix: "jobs[task_key='my_task'].notebook_task",
		},
		{
			input:      "resources.jobs[task_key='my_task'].notebook_task",
			n:          2,
			prefix:     "resources.jobs",
			skipPrefix: "[task_key='my_task'].notebook_task",
		},
		{
			input:      "resources.jobs[task_key='my_task'].notebook_task",
			n:          3,
			prefix:     "resources.jobs[task_key='my_task']",
			skipPrefix: "notebook_task",
		},
		{
			input:      "resources.jobs[task_key='my_task'].notebook_task",
			n:          4,
			prefix:     "resources.jobs[task_key='my_task'].notebook_task",
			skipPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			path, err := ParsePath(tt.input)
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
			path, err := ParsePath(tt.input)
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

		// Exact component matching - array indices, bracket keys, and key-value notation
		{
			name:     "prefix longer than path",
			s:        "a.b",
			prefix:   "a.b.c",
			expected: false,
		},
		{
			name:     "different array indices",
			s:        "items[0].name",
			prefix:   "items[1]",
			expected: false,
		},
		{
			name:     "different bracket keys",
			s:        "config['spark.conf']",
			prefix:   "config['other.conf']",
			expected: false,
		},
		{
			name:     "key-value in prefix",
			s:        "tasks[task_key='my_task'].notebook_task.source",
			prefix:   "tasks[task_key='my_task']",
			expected: true,
		},
		{
			name:     "different key-value values",
			s:        "tasks[task_key='my_task']",
			prefix:   "tasks[task_key='other_task']",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := ParsePath(tt.s)
			require.NoError(t, err)

			prefix, err := ParsePath(tt.prefix)
			require.NoError(t, err)

			result := path.HasPrefix(prefix)
			assert.Equal(t, tt.expected, result, "path.HasPrefix(prefix) where path=%q, prefix=%q", tt.s, tt.prefix)
		})
	}
}

func TestPathNodeYAMLMarshal(t *testing.T) {
	tests := []struct {
		name     string
		node     *PathNode
		expected string
	}{
		{
			name:     "simple field",
			node:     NewDotString(nil, "name"),
			expected: "name\n",
		},
		{
			name:     "nested path",
			node:     NewDotString(NewDotString(nil, "config"), "database"),
			expected: "config.database\n",
		},
		{
			name:     "path with array index",
			node:     NewDotString(NewIndex(NewDotString(nil, "items"), 0), "name"),
			expected: "items[0].name\n",
		},
		{
			name:     "path with key-value",
			node:     NewDotString(NewKeyValue(NewDotString(nil, "tags"), "key", "value"), "name"),
			expected: "tags[key='value'].name\n",
		},
		{
			name:     "nil path",
			node:     nil,
			expected: "null\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.node)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestPathNodeYAMLUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple field",
			input:    "name",
			expected: "name",
		},
		{
			name:     "nested path",
			input:    "config.database.host",
			expected: "config.database.host",
		},
		{
			name:     "path with array index",
			input:    "items[0].name",
			expected: "items[0].name",
		},
		{
			name:     "path with key-value",
			input:    "tags[key='server']",
			expected: "tags[key='server']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node PathNode
			err := yaml.Unmarshal([]byte(tt.input), &node)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, node.String())
		})
	}
}

// TestPathNodeYAMLNullAndEmpty tests YAML null and empty string handling.
func TestPathNodeYAMLNullAndEmpty(t *testing.T) {
	type Config struct {
		Path *PathNode `yaml:"path"`
	}

	// Null results in nil pointer (YAML doesn't call UnmarshalYAML for null)
	var config Config
	err := yaml.Unmarshal([]byte("path: null"), &config)
	require.NoError(t, err)
	assert.Nil(t, config.Path)

	// Empty string results in allocated pointer with zero-value PathNode.
	// The zero value has index=0, which represents "[0]" (array index 0).
	// This is a quirk - in practice, use null for "no path" in YAML configs.
	var config2 Config
	err = yaml.Unmarshal([]byte("path: ''"), &config2)
	require.NoError(t, err)
	require.NotNil(t, config2.Path)
	assert.Equal(t, "[0]", config2.Path.String())
}

func TestPathNodeYAMLUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		error string
	}{
		{
			name:  "unclosed bracket",
			input: "field[0",
			error: "unexpected end of input while parsing index",
		},
		{
			name:  "invalid character",
			input: "field..name",
			error: "expected field name after '.' but got '.' at position 6",
		},
		{
			name:  "unclosed quote",
			input: "field['key",
			error: "unexpected end of input while parsing map key",
		},
		{
			name:  "wildcard not allowed in PathNode",
			input: "tasks[*].name",
			error: "wildcards not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node PathNode
			err := yaml.Unmarshal([]byte(tt.input), &node)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.error)
		})
	}
}

// TestPathNodeYAMLRoundtrip tests that marshalling and unmarshalling preserves the path.
func TestPathNodeYAMLRoundtrip(t *testing.T) {
	paths := []string{
		"name",
		"config.database",
		"items[0].name",
		"tags[key='env'].value",
		"resources.jobs['my-job'].tasks[0]",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			// Parse -> Marshal -> Unmarshal -> compare
			original, err := ParsePath(path)
			require.NoError(t, err)

			data, err := yaml.Marshal(original)
			require.NoError(t, err)

			var restored PathNode
			err = yaml.Unmarshal(data, &restored)
			require.NoError(t, err)

			assert.Equal(t, path, restored.String())
		})
	}
}

// TestPathNodeYAMLInStruct tests PathNode as a field in a struct.
func TestPathNodeYAMLInStruct(t *testing.T) {
	type Config struct {
		Paths []*PathNode `yaml:"paths"`
	}

	yamlInput := `
paths:
  - name
  - config.database
  - items[0].value
  - tags[key='env']
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlInput), &config)
	require.NoError(t, err)
	require.Len(t, config.Paths, 4)

	assert.Equal(t, "name", config.Paths[0].String())
	assert.Equal(t, "config.database", config.Paths[1].String())
	assert.Equal(t, "items[0].value", config.Paths[2].String())
	assert.Equal(t, "tags[key='env']", config.Paths[3].String())
}

// TestPathNodeYAMLInStructWithErrors tests that invalid paths in YAML cause errors.
func TestPathNodeYAMLInStructWithErrors(t *testing.T) {
	type Config struct {
		Paths []*PathNode `yaml:"paths"`
	}

	yamlInput := `
paths:
  - name
  - field[invalid
  - config.database
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlInput), &config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of input")
}
