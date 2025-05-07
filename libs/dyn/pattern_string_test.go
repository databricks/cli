package dyn_test

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestNewPatternFromString(t *testing.T) {
	for ind, tc := range []struct {
		input  string
		output Pattern
		err    error
	}{
		{
			input:  "",
			output: NewPattern(),
		},
		{
			input:  ".",
			output: NewPattern(),
		},
		{
			input:  "foo.bar",
			output: NewPattern(Key("foo"), Key("bar")),
		},
		{
			input:  "[1]",
			output: NewPattern(Index(1)),
		},
		{
			input:  "foo[1].bar",
			output: NewPattern(Key("foo"), Index(1), Key("bar")),
		},
		{
			input:  "foo.bar[1]",
			output: NewPattern(Key("foo"), Key("bar"), Index(1)),
		},
		{
			input:  "foo.bar[1][2]",
			output: NewPattern(Key("foo"), Key("bar"), Index(1), Index(2)),
		},
		{
			input:  "foo.bar[1][2][3]",
			output: NewPattern(Key("foo"), Key("bar"), Index(1), Index(2), Index(3)),
		},
		{
			input:  "foo[1234]",
			output: NewPattern(Key("foo"), Index(1234)),
		},
		{
			input: "foo[123",
			err:   errors.New("invalid pattern: foo[123"),
		},
		{
			input: "foo[123]]",
			err:   errors.New("invalid pattern: foo[123]]"),
		},
		{
			input: "foo[[123]",
			err:   errors.New("invalid pattern: foo[[123]"),
		},
		{
			input: "foo[[123]]",
			err:   errors.New("invalid pattern: foo[[123]]"),
		},
		{
			input: "foo[foo]",
			err:   errors.New("invalid pattern: foo[foo]"),
		},
		{
			input: "foo..bar",
			err:   errors.New("invalid pattern: foo..bar"),
		},
		{
			input: "foo.bar.",
			err:   errors.New("invalid pattern: foo.bar."),
		},
		{
			// Every component may have a leading dot.
			input:  ".foo.[1].bar",
			output: NewPattern(Key("foo"), Index(1), Key("bar")),
		},
		{
			// But after an index there must be a dot.
			input: "foo[1]bar",
			err:   errors.New("invalid pattern: foo[1]bar"),
		},
		// Wildcard tests
		{
			input:  "foo.*",
			output: NewPattern(Key("foo"), AnyKey()),
		},
		{
			input:  "foo.*.bar",
			output: NewPattern(Key("foo"), AnyKey(), Key("bar")),
		},
		{
			input:  "foo[*]",
			output: NewPattern(Key("foo"), AnyIndex()),
		},
		{
			input:  "foo[*].bar",
			output: NewPattern(Key("foo"), AnyIndex(), Key("bar")),
		},
		{
			input:  "*[1]",
			output: NewPattern(AnyKey(), Index(1)),
		},
		{
			input:  "*.*",
			output: NewPattern(AnyKey(), AnyKey()),
		},
		{
			input:  "*[*]",
			output: NewPattern(AnyKey(), AnyIndex()),
		},
	} {
		t.Run(fmt.Sprintf("%d %s", ind, tc.input), func(t *testing.T) {
			p, err := NewPatternFromString(tc.input)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), tc.input)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.output, p)
			}
		})
	}
}

func TestMustPatternFromString(t *testing.T) {
	// Test valid pattern
	p := MustPatternFromString("foo.bar")
	assert.Equal(t, NewPattern(Key("foo"), Key("bar")), p)

	// Test with wildcards
	p = MustPatternFromString("foo.*.bar[*]")
	assert.Equal(t, NewPattern(Key("foo"), AnyKey(), Key("bar"), AnyIndex()), p)

	// Test that invalid pattern panics
	assert.Panics(t, func() {
		MustPatternFromString("foo[")
	})
}
