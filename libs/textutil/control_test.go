package textutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeControl(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		expected   string
		keepColors string
	}{
		{name: "empty"},
		{name: "plain", input: "plain text", expected: "plain text"},
		{name: "non-ASCII", input: "héllo — 日本語 ✓", expected: "héllo — 日本語 ✓"},
		{name: "newline and tab", input: "a\tb\nc", expected: "a\tb\nc"},
		{name: "CRLF", input: "a\r\nb\r\n", expected: "a\r\nb\r\n"},
		{name: "lone CR", input: "real\rfake", expected: `real\rfake`},
		{name: "CR at the end", input: "a\r", expected: `a\r`},
		{name: "OSC title with BEL", input: "a\x1b]0;title\x07b", expected: `a\x1b]0;title\ab`},
		{name: "OSC with ST", input: "\x1b]52;c;Zm9v\x1b\\", expected: `\x1b]52;c;Zm9v\x1b\`},
		{name: "CSI clear screen", input: "x\x1b[2Jy", expected: `x\x1b[2Jy`},
		{name: "SGR", input: "\x1b[31mred\x1b[0m", expected: `\x1b[31mred\x1b[0m`, keepColors: "\x1b[31mred\x1b[0m"},
		{name: "SGR with subparameters", input: "\x1b[38;2;1;2;3m\x1b[4:3m", expected: `\x1b[38;2;1;2;3m\x1b[4:3m`, keepColors: "\x1b[38;2;1;2;3m\x1b[4:3m"},
		{name: "SGR next to other sequences", input: "\x1b[1mb\x1b[2J\x1b]8;;http://x\x07", expected: `\x1b[1mb\x1b[2J\x1b]8;;http://x\a`, keepColors: "\x1b[1mb" + `\x1b[2J\x1b]8;;http://x\a`},
		{name: "unterminated SGR", input: "\x1b[31", expected: `\x1b[31`},
		{name: "private CSI ending in m", input: "\x1b[?25m", expected: `\x1b[?25m`},
		{name: "trailing ESC", input: "a\x1b", expected: `a\x1b`},
		{name: "NUL, BS, DEL", input: "a\x00b\x08c\x7f", expected: `a\x00b\bc\x7f`},
		{name: "C1 as UTF-8", input: "a\u009b2Jb\u0085", expected: `a\u009b2Jb\u0085`},
		{name: "raw C1 byte", input: "a\x9b2Jb", expected: `a\x9b2Jb`},
		{name: "truncated UTF-8", input: "a\xe6\x97", expected: `a\xe6\x97`},
		{name: "replacement character", input: "a�b", expected: "a�b"},
		{name: "backslash", input: `a\x1bb`, expected: `a\x1bb`},
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
