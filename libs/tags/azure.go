package tags

import (
	"regexp"

	"golang.org/x/text/unicode/rangetable"
)

// All characters that may not be used in Azure tag keys.
var azureForbiddenChars = rangetable.New('<', '>', '*', '&', '%', ';', '\\', '/', '+', '?')

var azureTag = &tag{
	keyLength:  512,
	keyPattern: regexp.MustCompile(`^[^<>\*&%;\\\/\+\?]*$`),
	keyNormalize: chain(
		replaceNotIn(latin1, '_'),
		replaceIn(azureForbiddenChars, '_'),
	),

	valueLength:  256,
	valuePattern: regexp.MustCompile(`^.*$`),
	valueNormalize: chain(
		replaceNotIn(latin1, '_'),
	),
}
