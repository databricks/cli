package run

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNopArgsHandler(t *testing.T) {
	h := nopArgsHandler{}
	opts := &Options{}

	// No error if no positional arguments are passed.
	err := h.ParseArgs([]string{}, opts)
	assert.NoError(t, err)

	// Error if any positional arguments are passed.
	err = h.ParseArgs([]string{"foo"}, opts)
	assert.EqualError(t, err, "received 1 unexpected positional arguments")

	// No completions.
	completions, _ := h.CompleteArgs([]string{}, "")
	assert.Nil(t, completions)
}

func TestArgsToKeyValueMap(t *testing.T) {
	for _, tc := range []struct {
		input    []string
		expected map[string]string
		tail     []string
		err      error
	}{
		{
			input:    []string{},
			expected: map[string]string{},
			tail:     []string{},
		},
		{
			input: []string{"--foo=bar", "--baz", "qux"},
			expected: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
			tail: []string{},
		},
		{
			input: []string{"--foo=bar", "--baz", "qux", "tail"},
			expected: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
			tail: []string{"tail"},
		},
		{
			input: []string{"--foo=bar", "--baz", "qux", "tail", "--foo=bar"},
			expected: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
			tail: []string{"tail", "--foo=bar"},
		},
		{
			input: []string{"--foo=bar", "--baz=qux"},
			expected: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
			tail: []string{},
		},
		{
			input: []string{"--foo=bar", "--baz=--qux"},
			expected: map[string]string{
				"foo": "bar",
				"baz": "--qux",
			},
			tail: []string{},
		},
		{
			input: []string{"--foo=bar", "--baz="},
			expected: map[string]string{
				"foo": "bar",
				"baz": "",
			},
			tail: []string{},
		},
		{
			input: []string{"--foo=bar", "--baz"},
			expected: map[string]string{
				"foo": "bar",
			},
			tail: []string{"--baz"},
		},
	} {
		actual, tail := argsToKeyValueMap(tc.input)
		assert.Equal(t, tc.expected, actual)
		assert.Equal(t, tc.tail, tail)
	}
}

func TestGenericParseKeyValueArgs(t *testing.T) {
	kv, err := genericParseKeyValueArgs([]string{"--foo=bar", "--baz", "qux"})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"foo": "bar",
		"baz": "qux",
	}, kv)

	_, err = genericParseKeyValueArgs([]string{"--foo=bar", "--baz", "qux", "tail"})
	assert.EqualError(t, err, "received 1 unexpected positional arguments")
}

func TestGenericCompleteKeyValueArgs(t *testing.T) {
	var completions []string

	// Complete nothing if there are no options.
	completions, _ = genericCompleteKeyValueArgs([]string{}, ``, []string{})
	assert.Empty(t, completions)

	// Complete nothing if we're in the middle of a key-value pair (as single argument with equals sign).
	completions, _ = genericCompleteKeyValueArgs([]string{}, `--foo=`, []string{`foo`, `bar`})
	assert.Empty(t, completions)

	// Complete nothing if we're in the middle of a key-value pair (as two arguments).
	completions, _ = genericCompleteKeyValueArgs([]string{`--foo`}, ``, []string{`foo`, `bar`})
	assert.Empty(t, completions)

	// Complete if we're at the beginning.
	completions, _ = genericCompleteKeyValueArgs([]string{}, ``, []string{`foo`, `bar`})
	assert.Equal(t, []string{`--foo=`, `--bar=`}, completions)

	// Complete if we have already one key-value pair.
	completions, _ = genericCompleteKeyValueArgs([]string{`--foo=bar`}, ``, []string{`foo`, `bar`})
	assert.Equal(t, []string{`--bar=`}, completions)
}
