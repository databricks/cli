package tags

import (
	"regexp"
	"unicode"

	"golang.org/x/text/unicode/rangetable"
)

// The union of all characters allowed in AWS tags.
// This must be used only after filtering out non-Latin1 characters,
// because the [unicode] classes include non-Latin1 characters.
var awsChars = rangetable.Merge(
	unicode.Digit,
	unicode.Space,
	unicode.Letter,
	rangetable.New('+', '-', '=', '.', ':', '/', '@'),
)

var awsTag = &tag{
	keyLength:  127,
	keyPattern: regexp.MustCompile(`^[\d \w\+\-=\.:\/@]*$`),
	keyNormalize: chain(
		normalizeMarks(),
		replaceNotIn(latin1, '_'),
		replaceNotIn(awsChars, '_'),
	),

	valueLength:  255,
	valuePattern: regexp.MustCompile(`^[\d \w\+\-=\.:/@]*$`),
	valueNormalize: chain(
		normalizeMarks(),
		replaceNotIn(latin1, '_'),
		replaceNotIn(awsChars, '_'),
	),
}
