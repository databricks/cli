package textutil

import (
	"strings"
	"unicode"
)

// We leave the full range of unicode letters in tact, but remove all "special" characters,
// including spaces and dots, which are not supported in e.g. experiment names or YAML keys.
func NormaliseString(name string) string {
	name = strings.ToLower(name)
	return strings.Map(replaceNonAlphanumeric, name)
}

func replaceNonAlphanumeric(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	return '_'
}
