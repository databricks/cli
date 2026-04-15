package dyn_test

import (
	"errors"
	"testing"

	. "github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestPathStringRoundTrip(t *testing.T) {
	for _, tc := range []Path{
		NewPath(Key("foo"), Key("bar")),
		NewPath(Key("foo"), Index(1), Key("bar")),
		NewPath(Key("configuration"), Key("europris.swipe.egress_streaming_schema")),
		NewPath(Key("foo"), Key("it's.here")),
		NewPath(Key("a.b"), Key("c"), Index(0)),
	} {
		s := tc.String()
		p, err := NewPathFromString(s)
		assert.NoError(t, err, s)
		assert.Equal(t, tc, p, s)
	}
}

func TestNewPathFromString(t *testing.T) {
	for _, tc := range []struct {
		input  string
		output Path
		err    error
	}{
		{
			input:  "",
			output: NewPath(),
		},
		{
			input:  ".",
			output: NewPath(),
		},
		{
			input:  "foo.bar",
			output: NewPath(Key("foo"), Key("bar")),
		},
		{
			input:  "[1]",
			output: NewPath(Index(1)),
		},
		{
			input:  "foo[1].bar",
			output: NewPath(Key("foo"), Index(1), Key("bar")),
		},
		{
			input:  "foo.bar[1]",
			output: NewPath(Key("foo"), Key("bar"), Index(1)),
		},
		{
			input:  "foo.bar[1][2]",
			output: NewPath(Key("foo"), Key("bar"), Index(1), Index(2)),
		},
		{
			input:  "foo.bar[1][2][3]",
			output: NewPath(Key("foo"), Key("bar"), Index(1), Index(2), Index(3)),
		},
		{
			input:  "foo[1234]",
			output: NewPath(Key("foo"), Index(1234)),
		},
		{
			input: "foo[123",
			err:   errors.New("invalid path: foo[123"),
		},
		{
			input: "foo[123]]",
			err:   errors.New("invalid path: foo[123]]"),
		},
		{
			input: "foo[[123]",
			err:   errors.New("invalid path: foo[[123]"),
		},
		{
			input: "foo[[123]]",
			err:   errors.New("invalid path: foo[[123]]"),
		},
		{
			input: "foo[foo]",
			err:   errors.New("invalid path: foo[foo]"),
		},
		{
			input: "foo..bar",
			err:   errors.New("invalid path: foo..bar"),
		},
		{
			input: "foo.bar.",
			err:   errors.New("invalid path: foo.bar."),
		},
		{
			// Every component may have a leading dot.
			input:  ".foo.[1].bar",
			output: NewPath(Key("foo"), Index(1), Key("bar")),
		},
		{
			// But after an index there must be a dot.
			input: "foo[1]bar",
			err:   errors.New("invalid path: foo[1]bar"),
		},
		{
			// * is parsed as regular string in NewPathFromString
			input:  "foo.*",
			output: NewPath(Key("foo"), Key("*")),
		},
		{
			// * is parsed as regular string in NewPathFromString
			input:  "foo.*.bar",
			output: NewPath(Key("foo"), Key("*"), Key("bar")),
		},
		{
			// This is an invalid path (but would be valid for patterns)
			input: "foo[*].bar",
			err:   errors.New("invalid path: foo[*].bar"),
		},
		{
			// Bracket notation with quoted key containing dots.
			input:  "foo['bar.baz']",
			output: NewPath(Key("foo"), Key("bar.baz")),
		},
		{
			// Bracket notation at the start.
			input:  "['a.b'].foo",
			output: NewPath(Key("a.b"), Key("foo")),
		},
		{
			// Bracket notation with escaped single quote.
			input:  "foo['it''s.here']",
			output: NewPath(Key("foo"), Key("it's.here")),
		},
		{
			// Bracket notation followed by index.
			input:  "foo['bar.baz'][0]",
			output: NewPath(Key("foo"), Key("bar.baz"), Index(0)),
		},
		{
			// Unterminated bracket notation.
			input: "foo['bar",
			err:   errors.New("invalid path: foo['bar"),
		},
		{
			// Missing closing bracket.
			input: "foo['bar'",
			err:   errors.New("invalid path: foo['bar'"),
		},
	} {
		p, err := NewPathFromString(tc.input)
		if tc.err != nil {
			assert.EqualError(t, err, tc.err.Error(), tc.input)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.output, p)
		}
	}
}
