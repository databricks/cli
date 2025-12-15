package shellquote

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashArg(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Simple cases - no quoting needed
		{"hello", "hello"},
		{"hello-world", "hello-world"},
		{"hello_world", "hello_world"},
		{"path/to/file", "path/to/file"},
		{"file.txt", "file.txt"},
		{"host:port", "host:port"},
		{"123", "123"},
		{"a1b2c3", "a1b2c3"},

		// Empty string
		{"", "''"},

		// Cases requiring quoting - spaces
		{"hello world", "'hello world'"},
		{"Custom Job Name", "'Custom Job Name'"},

		// Cases requiring quoting - special characters
		{"job_name=Custom Job Name", "'job_name=Custom Job Name'"},
		{"foo=bar", "'foo=bar'"},
		{"a b c", "'a b c'"},
		{"*", "'*'"},
		{"$VAR", "'$VAR'"},
		{"a|b", "'a|b'"},
		{"a&b", "'a&b'"},
		{"a;b", "'a;b'"},
		{"a>b", "'a>b'"},
		{"a<b", "'a<b'"},
		{"a(b)", "'a(b)'"},
		{"a[b]", "'a[b]'"},
		{"a{b}", "'a{b}'"},
		{"a`b`", "'a`b`'"},
		{"a\\b", "'a\\b'"},
		{"a\"b\"", `'a"b"'`},

		// Single quotes in string
		{"it's", `'it'\''s'`},
		{"can't", `'can'\''t'`},
		{"'", `''\'''`},
		{"''", `''\'''\'''`},
		{"a'b'c", `'a'\''b'\''c'`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := BashArg(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
