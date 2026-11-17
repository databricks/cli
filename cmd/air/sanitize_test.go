package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripControl(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain text is unchanged", "sunny-cat-42", "sunny-cat-42"},
		{"non-ASCII text is kept", "résumé 日本語 🚀", "résumé 日本語 🚀"},
		{"OSC set-title terminated by BEL", "\x1b]0;pwned\x07name", "name"},
		{"OSC set-title terminated by ST", "\x1b]0;pwned\x1b\\name", "name"},
		{"OSC 52 clipboard write", "a\x1b]52;c;cHduZWQ=\x07b", "ab"},
		{"CSI clear screen", "a\x1b[2Jb", "ab"},
		{"CSI color", "\x1b[31mred\x1b[0m", "red"},
		{"carriage return", "real-name\rfake", "real-namefake"},
		{"tab and newline", "a\tb\nc", "abc"},
		{"DEL", "a\x7fb", "ab"},
		{"C1 control", "a\u0085b\u009bc", "abc"},
		{"two-byte escape sequence", "a\x1bbz", "az"},
		{"trailing ESC", "a\x1b", "a"},
		{"NUL", "a\x00b", "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stripControl(tt.in))
		})
	}
}
