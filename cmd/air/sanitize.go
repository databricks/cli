package aircmd

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/muesli/termenv"
)

// osc8 wraps label in an OSC 8 terminal hyperlink to url. A control character
// in url (or a byte that is not valid UTF-8, which can be one to the terminal)
// would end the sequence early and let the rest of url act on the terminal, so
// such a url gets no link.
func osc8(url, label string) string {
	if !utf8.ValidString(url) || strings.ContainsFunc(url, unicode.IsControl) {
		return label
	}
	return termenv.Hyperlink(url, label)
}
