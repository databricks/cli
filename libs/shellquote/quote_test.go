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

func TestWindowsArg(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Backslashes and 8.3 short names are not special to cmd.exe or PowerShell.
		{`C:\Users\me\skills`, `C:\Users\me\skills`},
		{`C:\Users\RUNNER~1\AppData\Local\Temp\out`, `C:\Users\RUNNER~1\AppData\Local\Temp\out`},
		{`.\out`, `.\out`},

		// cmd.exe splits at spaces and command separators.
		{`C:\my skills`, `"C:\my skills"`},
		{`C:\Program Files (x86)\out`, `"C:\Program Files (x86)\out"`},
		// PowerShell also splits at Unicode spaces such as a no-break space.
		{"C:\\my\u00a0skills", "\"C:\\my\u00a0skills\""},
		{"C:\\my\u3000skills", "\"C:\\my\u3000skills\""},
		{`C:\a&b`, `"C:\a&b"`},
		{`C:\a;b`, `"C:\a;b"`},
		// PowerShell starts a string at a bare single quote, a script block at a
		// brace, and splats or comments at @ and #.
		{`C:\it's`, `"C:\it's"`},
		{"C:\\it\u2019s", "\"C:\\it\u2019s\""},
		{`C:\a{b}`, `"C:\a{b}"`},
		{`@out`, `"@out"`},
		{`C:\#out`, `"C:\#out"`},
		{`data science,databricks-jobs`, `"data science,databricks-jobs"`},

		// A trailing backslash would escape the closing quote.
		{`C:\my skills\`, `"C:\my skills\\"`},
		{`C:\my skills\\`, `"C:\my skills\\\\"`},

		{"", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := WindowsArg(tt.input)
			assert.True(t, ok)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestWindowsArgRejectsWhatNoQuotingKeepsLiteral(t *testing.T) {
	for _, input := range []string{
		// PowerShell expands variables and runs subexpressions inside double quotes.
		`C:\$cache\skills`,
		`$(Remove-Item -Recurse ~)`,
		"C:\\a`b",
		// cmd.exe expands %VAR% inside double quotes, and !VAR! with delayed
		// expansion.
		`C:\%TEMP%\skills`,
		`C:\!TEMP!\skills`,
		`C:\a!b`,
		// A double quote, or a typographic one PowerShell reads as one, ends the string.
		`C:\a"b`,
		"C:\\a\u201cb",
		// A line break ends the pasted command; other control characters do not
		// paste as typed.
		"C:\\a\nb",
		"C:\\a\r\ndel *",
		"C:\\a\rb",
		"C:\\a\tb",
	} {
		t.Run(input, func(t *testing.T) {
			_, ok := WindowsArg(input)
			assert.False(t, ok)
		})
	}
}
