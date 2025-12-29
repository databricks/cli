package shellquote

import (
	"strings"
)

// BashArg quotes a string for safe use as a bash command argument.
// It returns the argument unquoted if it contains only safe characters,
// otherwise it wraps it in single quotes and escapes any single quotes within.
func BashArg(s string) string {
	if s == "" {
		return "''"
	}

	// Check if string needs quoting
	needsQuoting := false
	for _, c := range s {
		if !isSafeChar(c) {
			needsQuoting = true
			break
		}
	}

	if !needsQuoting {
		return s
	}

	// Use single quotes and escape any single quotes in the string
	// by ending the quote, adding an escaped quote, and starting a new quote
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// isSafeChar returns true if the character is safe to use unquoted in bash.
func isSafeChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '/' || c == '.' || c == ':'
}
