package textutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeControl(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		// What EscapeControlKeepColors returns, if not the same.
		keepColors string
	}{
		{name: "empty"},
		{name: "plain", input: "plain text", expected: "plain text"},
		{name: "non-ASCII", input: "héllo — 日本語 ✓", expected: "héllo — 日本語 ✓"},
		{name: "newline and tab", input: "a\tb\nc", expected: "a\tb\nc"},
		{name: "CRLF", input: "a\r\nb\r\n", expected: "a\nb\n"},
		{name: "CRLF at the start", input: "\r\na", expected: "\na"},
		{name: "lone CR", input: "real\rfake", expected: `real\rfake`},
		{name: "CR at the end", input: "a\r", expected: `a\r`},
		{name: "CR LF", input: "a\r\r\nb", expected: `a\r` + "\nb"},
		{name: "OSC title with BEL", input: "a\x1b]0;title\x07b", expected: `a\u001b]0;title\u0007b`},
		{name: "OSC with ST", input: "\x1b]52;c;Zm9v\x1b\\", expected: `\u001b]52;c;Zm9v\u001b\`},
		{name: "CSI clear screen", input: "x\x1b[2Jy", expected: `x\u001b[2Jy`},
		{name: "SGR", input: "\x1b[31mred\x1b[0m", expected: `\u001b[31mred\u001b[0m`, keepColors: "\x1b[31mred\x1b[0m"},
		{name: "SGR with subparameters", input: "\x1b[38;2;1;2;3m\x1b[4:3mx\x1b[m", expected: `\u001b[38;2;1;2;3m\u001b[4:3mx\u001b[m`, keepColors: "\x1b[38;2;1;2;3m\x1b[4:3mx\x1b[m"},
		{name: "style left on", input: "a\x1b[0m\x1b[8mhidden\n", expected: `a\u001b[0m\u001b[8mhidden` + "\n", keepColors: "a\x1b[0m\x1b[8mhidden\x1b[0m\n"},
		{name: "style left on in a cell", input: "\x1b[8ma\tb\x1b[31m\nc", expected: `\u001b[8ma` + "\tb" + `\u001b[31m` + "\nc", keepColors: "\x1b[8ma\x1b[0m\tb\x1b[31m\x1b[0m\nc"},
		{name: "style closed before the line ends", input: "\x1b[32mok\x1b[0m\n\x1b[32mok\n\x1b[0m", expected: `\u001b[32mok\u001b[0m` + "\n" + `\u001b[32mok` + "\n" + `\u001b[0m`, keepColors: "\x1b[32mok\x1b[0m\n" + "\x1b[32mok\x1b[0m\n" + "\x1b[0m"},
		{name: "SGR next to other sequences", input: "\x1b[1mb\x1b[2J\x1b]8;;http://x\x07", expected: `\u001b[1mb\u001b[2J\u001b]8;;http://x\u0007`, keepColors: "\x1b[1mb" + `\u001b[2J\u001b]8;;http://x\u0007` + "\x1b[0m"},
		{name: "unterminated SGR", input: "\x1b[31", expected: `\u001b[31`},
		{name: "SGR cut by a newline", input: "\x1b[31\nm", expected: `\u001b[31` + "\nm"},
		{name: "private CSI ending in m", input: "\x1b[?25m", expected: `\u001b[?25m`},
		{name: "trailing ESC", input: "a\x1b", expected: `a\u001b`},
		{name: "NUL, BS, FF, DEL", input: "a\x00b\x08c\x0cd\x7f", expected: `a\u0000b\bc\fd\u007f`},
		{name: "C1 as UTF-8", input: "a\u009b2Jb\u0085", expected: `a\u009b2Jb\u0085`},
		{name: "raw C1 byte", input: "a\x9b2Jb", expected: `a�2Jb`},
		{name: "truncated UTF-8", input: "a\xe6\x97", expected: `a��`},
		{name: "replacement character", input: "a�b", expected: "a�b"},
		{name: "backslash", input: `a\u001bb`, expected: `a\u001bb`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, EscapeControl(c.input))
			keepColors := c.keepColors
			if keepColors == "" {
				keepColors = c.expected
			}
			assert.Equal(t, keepColors, EscapeControlKeepColors(c.input))
		})
	}
}

func TestEscapeControlOneLine(t *testing.T) {
	assert.Equal(t, "plain", EscapeControlOneLine("plain"))
	assert.Equal(t, `a\tb\nc\r\nd\u001b[2J\r`, EscapeControlOneLine("a\tb\nc\r\nd\x1b[2J\r"))
	// Text that is already escaped is unchanged.
	assert.Equal(t, `a\tb\u001b[2J`, EscapeControlOneLine(`a\tb\u001b[2J`))
}
