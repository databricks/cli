package dynvar

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefNoString(t *testing.T) {
	_, ok := NewRef(dyn.V(1))
	require.False(t, ok, "should not match non-string")
}

func TestNewRefSpans(t *testing.T) {
	// Spans must point at the unescaped occurrence, not the escaped one that
	// appears earlier with identical text.
	ref, ok := NewRef(dyn.V("$${a.b} ${a.b}"))
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
