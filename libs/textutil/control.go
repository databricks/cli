package textutil

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// EscapeControl returns s with terminal control characters replaced by the
// escape JSON output shows for them, such as `\u001b` or `\r`. Use it on text
// the CLI did not write itself (API responses and error messages, names of
// remote objects) before printing it as text, so that whoever chose the text
// cannot move the cursor, clear the screen, retitle the window, write the
// clipboard or overwrite a line of output on the terminal of whoever reads it.
//
// C0 and C1 control characters and DEL are escaped, and a byte that is not
// valid UTF-8 becomes `�`. Newline and tab are kept, and CRLF becomes a
// newline, so the result has no control character other than those two. The
// escape is visible rather than dropped so that an error naming an object shows
// the name it has. JSON output must not go through this: the JSON encoder has
// its own escaping.
func EscapeControl(s string) string {
	return escapeControl(s, noColors)
}

// EscapeControlOneLine is [EscapeControl] for a value shown in a one-line slot
// such as a table cell: line breaks and tabs are escaped too, because they
// would start a new row or column.
func EscapeControlOneLine(s string) string {
	if strings.ContainsAny(s, "\r\n\t") {
		s = strings.NewReplacer("\r", `\r`, "\n", `\n`, "\t", `\t`).Replace(s)
	}
	return escapeControl(s, noColors)
}

// EscapeControlKeepColors is [EscapeControl] for text the CLI composed from its
// own text and other values, where colors are on: SGR sequences (colors and
// text styles, ESC [ ... m) are kept, because the CLI's own styling is already
// part of the string. A style that is still on at the end is reset, so that a
// value cannot hide what is printed after it. Prefer EscapeControl on the
// individual values where the caller has them, and where the CLI adds no
// colors.
func EscapeControlKeepColors(s string) string {
	return escapeControl(s, keepColors)
}

// EscapeControlKeepColorsPerCell is [EscapeControlKeepColors] for a rendered
// table: a style that is still on at the end of a line or a tab-separated cell
// is reset there, so that a value cannot hide the next cell or row.
func EscapeControlKeepColorsPerCell(s string) string {
	return escapeControl(s, keepColorsPerCell)
}

type colorMode int

const (
	noColors colorMode = iota
	keepColors
	keepColorsPerCell
)

func escapeControl(s string, colors colorMode) string {
	var b strings.Builder
	// s[flushed:i] is text that needs no escaping and is not in b yet.
	flushed := 0
	// Whether the last SGR sequence kept turned a color or style on.
	styled := false
	changed := false
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		invalid := r == utf8.RuneError && size == 1
		keep := size
		switch {
		case invalid:
			keep = 0
		case (r == '\n' || r == '\t') && styled && colors == keepColorsPerCell:
			b.WriteString(s[flushed:i])
			b.WriteString(sgrReset)
			flushed = i
			styled = false
			changed = true
		case r == '\n' || r == '\t':
		case r == '\x1b' && colors != noColors:
			keep = sgrLen(s[i:])
			if keep > 0 {
				styled = s[i:i+keep] != sgrReset && s[i:i+keep] != "\x1b[m"
			}
		case unicode.IsControl(r):
			keep = 0
		}
		if keep > 0 {
			i += keep
			continue
		}

		b.WriteString(s[flushed:i])
		switch {
		case invalid:
			b.WriteString(`�`)
		case r == '\r' && strings.HasPrefix(s[i+1:], "\n"):
			// CRLF: only the newline that follows is kept.
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		default:
			fmt.Fprintf(&b, `\u%04x`, r)
		}
		i += size
		flushed = i
		changed = true
	}
	if !changed && !styled {
		return s
	}
	b.WriteString(s[flushed:])
	if styled {
		b.WriteString(sgrReset)
	}
	return b.String()
}

const sgrReset = "\x1b[0m"

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
