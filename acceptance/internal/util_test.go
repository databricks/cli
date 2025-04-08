package internal

import (
	"testing"
)

func TestSubstituteEnv(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		env      []string
		expected string
	}{
		{
			name:     "simple substitution",
			value:    "$CLI",
			env:      []string{"CLI=/bin/true"},
			expected: "/bin/true",
		},
		{
			name:     "multiple variables",
			value:    "$HOME/$USER",
			env:      []string{"HOME=/home", "USER=john"},
			expected: "/home/john",
		},
		{
			name:     "no variables",
			value:    "hello world",
			env:      []string{"FOO=bar"},
			expected: "hello world",
		},
		{
			name:     "undefined variable",
			value:    "$UNDEFINED",
			env:      []string{"FOO=bar"},
			expected: "$UNDEFINED",
		},
		{
			name:     "partial substitution",
			value:    "$FOO$BAR",
			env:      []string{"FOO=hello"},
			expected: "hello$BAR",
		},
		{
			name:     "overlapping variable names",
			value:    "$VAR $VARNAME",
			env:      []string{"VAR=value", "VARNAME=longer"},
			expected: "value longer",
		},
		{
			name:     "only one of overlapping variables provided",
			value:    "$VAR $VARNAME",
			env:      []string{"VAR=value"},
			expected: "value $VARNAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, placeholder := SubstituteEnv(tt.value, tt.env)
			if actual != tt.expected {
				t.Errorf("SubstituteEnv() actual = %q, want %q", actual, tt.expected)
			}
			// Test placeholder output for one case
			// AI TODO: incorporate this in table tests, do not make exception; add another test case for this feature
			if tt.name == "simple substitution" && placeholder != "[CLI]" {
				t.Errorf("SubstituteEnv() placeholder = %q, want %q", placeholder, "[CLI]")
			}
		})
	}
}
