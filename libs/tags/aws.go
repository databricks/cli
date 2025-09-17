package tags

import (
	"regexp"
	"unicode"

	"github.com/databricks/cli/libs/textutil"

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
	keyNormalize: textutil.Chain(
		textutil.NormalizeMarks(),
		textutil.ReplaceNotIn(textutil.Latin1, '_'),
		textutil.ReplaceNotIn(awsChars, '_'),
	),

	valueLength:  255,
	valuePattern: regexp.MustCompile(`^[\d \w\+\-=\.:/@]*$`),
	valueNormalize: textutil.Chain(
		textutil.NormalizeMarks(),
		textutil.ReplaceNotIn(textutil.Latin1, '_'),
		textutil.ReplaceNotIn(awsChars, '_'),
	),
}
