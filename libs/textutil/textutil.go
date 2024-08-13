package textutil

import (
	"regexp"
	"strings"
	"unicode"
)

// We leave the full range of unicode letters in tact, but remove all "special" characters,
// including spaces and dots, which are not supported in e.g. experiment names or YAML keys.
func NormalizeString(name string) string {
	name = strings.ToLower(name)
	s := strings.Map(replaceNonAlphanumeric, name)

	// replacing multiple underscores with a single one
	re := regexp.MustCompile(`_+`)
	s = re.ReplaceAllString(s, "_")

	// removing leading and trailing underscores
	return strings.Trim(s, "_")
}

func replaceNonAlphanumeric(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	return '_'
}
