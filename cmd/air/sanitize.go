package aircmd

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/databricks/cli/libs/textutil"
	"github.com/muesli/termenv"
)

// oneLine escapes a server-supplied string such as an MLflow run name for a
// one-line slot (a table cell, a box field): terminal control characters as
// [textutil.EscapeControl] does, and also line breaks and tabs, which would
// break the layout. Apply it to display values only: JSON output and link
// targets are left as the server sent them.
func oneLine(s string) string {
	if strings.ContainsAny(s, "\r\n\t") {
		s = strings.NewReplacer("\r", `\r`, "\n", `\n`, "\t", `\t`).Replace(s)
	}
	return textutil.EscapeControl(s)
}

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
