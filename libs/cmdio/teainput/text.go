package teainput

import "strings"

// FirstLine returns text up to its first line break (CR or LF). Inputs that
// hold one line, such as a search box, take a pasted text this way: a copied
// line often ends in a line break, which the input can't show or match.
func FirstLine(text string) string {
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		return text[:i]
	}
	return text
}
