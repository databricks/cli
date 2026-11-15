package shellquote

import (
	"strings"
	"unicode"
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

// WindowsArg quotes a string for use as a command argument that cmd.exe and
// PowerShell pass to the program as one argument, without expanding anything
// in it. It returns the argument unquoted unless it contains whitespace (any
// Unicode space: PowerShell splits arguments at a no-break space too), a
// cmd.exe command separator, or a character PowerShell treats specially outside
// quotes (a single quote starts a string, braces a script block, @ and #
// splatting and comments); backslashes are path separators there, not escapes.
// Quoted arguments get their trailing backslashes doubled, so the closing
// quote is not escaped when the program parses its command line from cmd.exe.
// PowerShell 7.3 and later quote arguments themselves and treat a backslash as
// a literal, so there a doubled trailing backslash reaches the program as two
// backslashes; for a path that is harmless.
//
// Parsing rules encoded here:
//   - Program command lines: https://learn.microsoft.com/en-us/cpp/c-language/parsing-c-command-line-arguments
//   - cmd.exe: https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/cmd
//   - PowerShell quoting: https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_quoting_rules
//   - PowerShell native argument passing: https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_parsing
//
// The second result is false when no quoting keeps s literal in both shells:
// inside double quotes PowerShell still expands $ and ` (a $(...) runs a
// command), cmd.exe still expands %VAR% and, with delayed expansion, !VAR!, a
// double quote ends the string, and a control character such as a line break
// ends the command when pasted. Callers must not print such a string as part
// of a command.
func WindowsArg(s string) (string, bool) {
	if strings.ContainsAny(s, "$`%!\"“”„") || strings.ContainsFunc(s, unicode.IsControl) {
		return "", false
	}
	if s == "" {
		return `""`, true
	}
	if !strings.ContainsAny(s, "&^()<>|,;='{}@#‘’‚‛") && !strings.ContainsFunc(s, unicode.IsSpace) {
		return s, true
	}
	trailing := len(s) - len(strings.TrimRight(s, `\`))
	return `"` + s + strings.Repeat(`\`, trailing) + `"`, true
}
