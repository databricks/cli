package textutil

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// EscapeControl returns s with terminal control characters replaced by the
// escape a Go string literal would use, such as `\x1b` or `\r`. Use it on text the CLI did not write itself
// (API responses and error messages, names of remote objects) before printing
// it as text, so that whoever chose the text cannot move the cursor, clear the
// screen, retitle the window, write the clipboard or overwrite a line of output
// on the terminal of whoever reads it.
//
// C0 and C1 control characters, DEL and bytes that are not valid UTF-8 are
// escaped. Newline and tab are kept, and so is a carriage return that is part
// of a CRLF line ending. The escape is visible rather than dropped so that two
// different values never print the same. JSON output must not go through this:
// the JSON encoder has its own escaping.
func EscapeControl(s string) string {
	return escapeControl(s, false)
}

// EscapeControlKeepColors is [EscapeControl] for a line the CLI composed from
// its own text and other values: SGR sequences (colors and text styles, ESC [
// ... m) are kept, because the CLI's own styling is already part of the string.
// Prefer EscapeControl on the individual values where the caller has them.
func EscapeControlKeepColors(s string) string {
	return escapeControl(s, true)
}

func escapeControl(s string, keepColors bool) string {
	var b strings.Builder
	// s[flushed:i] is text that needs no escaping and is not in b yet.
	flushed := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		invalid := r == utf8.RuneError && size == 1
		if !invalid {
			keep := size
			switch {
			case r == '\n' || r == '\t':
			case r == '\r' && strings.HasPrefix(s[i+1:], "\n"):
			case r == '\x1b' && keepColors:
				keep = sgrLen(s[i:])
			case unicode.IsControl(r):
				keep = 0
			}
			if keep > 0 {
				i += keep
				continue
			}
		}
		b.WriteString(s[flushed:i])
		if invalid {
			fmt.Fprintf(&b, `\x%02x`, s[i])
		} else {
			quoted := strconv.QuoteRuneToASCII(r)
			b.WriteString(quoted[1 : len(quoted)-1])
		}
		i += size
		flushed = i
	}
	if flushed == 0 {
		return s
	}
	b.WriteString(s[flushed:])
	return b.String()
}

// sgrLen returns the length of the SGR sequence (ESC [ parameters m) that s
// starts with, or 0 if s starts with anything else.
func sgrLen(s string) int {
	if !strings.HasPrefix(s, "\x1b[") {
		return 0
	}
	for i := 2; i < len(s); i++ {
		c := s[i]
		if c == 'm' {
			return i + 1
		}
		if (c < '0' || c > '9') && c != ';' && c != ':' {
			return 0
		}
	}
	return 0
}
