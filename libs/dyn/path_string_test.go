package dyn_test

import (
	"fmt"
	"testing"

	. "github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
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
			err:   fmt.Errorf("invalid path: foo[123"),
		},
		{
			input: "foo[123]]",
			err:   fmt.Errorf("invalid path: foo[123]]"),
		},
		{
			input: "foo[[123]",
			err:   fmt.Errorf("invalid path: foo[[123]"),
		},
		{
			input: "foo[[123]]",
			err:   fmt.Errorf("invalid path: foo[[123]]"),
		},
		{
			input: "foo[foo]",
			err:   fmt.Errorf("invalid path: foo[foo]"),
		},
		{
			input: "foo..bar",
			err:   fmt.Errorf("invalid path: foo..bar"),
		},
		{
			input: "foo.bar.",
			err:   fmt.Errorf("invalid path: foo.bar."),
		},
		{
			// Every component may have a leading dot.
			input:  ".foo.[1].bar",
			output: NewPath(Key("foo"), Index(1), Key("bar")),
		},
		{
			// But after an index there must be a dot.
			input: "foo[1]bar",
			err:   fmt.Errorf("invalid path: foo[1]bar"),
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
