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
		// AI TODO with overlapping names $VAR $VARVAR; test what happens when only one is provided
		// AI TODO: fix replacement to match full words to handle this case correctly
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SubstituteEnv(tt.value, tt.env)
			if result != tt.expected {
				t.Errorf("SubstituteEnv() = %q, want %q", result, tt.expected)
			}
		})
	}
}
