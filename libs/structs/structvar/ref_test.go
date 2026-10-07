package structvar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefSpans(t *testing.T) {
	// Spans must point at the unescaped occurrence, not the escaped one that
	// appears earlier with identical text.
	ref, ok := NewRef("$${a.b} ${a.b}")
	require.True(t, ok)
	require.Len(t, ref.Matches, 1)
	require.Len(t, ref.Spans, 1)
	assert.Equal(t, "${a.b}", ref.Str[ref.Spans[0][0]:ref.Spans[0][1]])
	assert.Equal(t, 8, ref.Spans[0][0])
}

func TestContainsVariableReference(t *testing.T) {
	tests := map[string]bool{
		"${a.b}":          true,
		"x ${a.b} y":      true,
		"$${a.b}":         false,
		"$${a.b} $${c.d}": false,
		"$${a.b} ${c.d}":  true,
		"no refs":         false,
		"":                false,
	}
	for in, want := range tests {
		assert.Equal(t, want, ContainsVariableReference(in), "input %q", in)
	}
}

func TestUnescape(t *testing.T) {
	assert.Equal(t, "${a.b}", Unescape("$${a.b}"))
	assert.Equal(t, "${a} ${b}", Unescape("$${a} $${b}"))
	assert.Equal(t, "${a.b}", Unescape("${a.b}"), "already unescaped is unchanged")
	assert.Equal(t, "no refs", Unescape("no refs"))
}

func TestReplaceRef(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"${a.b}", "V"},
		{"$${a.b}", "$${a.b}"},
		{"$${a.b} ${a.b}", "$${a.b} V"},
		{"${a.b} $${a.b}", "V $${a.b}"},
		{"${a.b}-${a.b}", "V-V"},
		{"$${a.b} ${a.b} $${a.b}", "$${a.b} V $${a.b}"},
		{"untouched ${c.d}", "untouched ${c.d}"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ReplaceRef(tt.in, "${a.b}", "V"), "input %q", tt.in)
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
