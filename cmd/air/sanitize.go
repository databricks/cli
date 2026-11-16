package aircmd

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// stripControl removes terminal escape sequences (CSI, OSC, DCS, ...) and C0/C1
// control characters from s, so a server-supplied string such as an MLflow run
// name can be shown without letting whoever chose it clear the screen, retitle
// the window, write the clipboard or overwrite a line of output. Printable text,
// including non-ASCII, is kept. Apply it to display values only: JSON output and
// link targets are left as the server sent them.
func stripControl(s string) string {
	// Drop whole escape sequences first, so "\x1b[2J" does not leave "[2J"
	// behind once the ESC is removed below.
	s = ansi.Strip(s)
	if !strings.ContainsFunc(s, unicode.IsControl) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
