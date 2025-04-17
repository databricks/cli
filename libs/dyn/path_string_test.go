package dyn_test

import (
	"errors"
	"testing"

	. "github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

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
