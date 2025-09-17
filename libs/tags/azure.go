package tags

import (
	"regexp"

	"github.com/databricks/cli/libs/textutil"

	"golang.org/x/text/unicode/rangetable"
)

// All characters that may not be used in Azure tag keys.
var azureForbiddenChars = rangetable.New('<', '>', '*', '&', '%', ';', '\\', '/', '+', '?')

var azureTag = &tag{
	keyLength:  512,
	keyPattern: regexp.MustCompile(`^[^<>\*&%;\\\/\+\?]*$`),
	keyNormalize: textutil.Chain(
		textutil.ReplaceNotIn(latin1, '_'),
		textutil.ReplaceIn(azureForbiddenChars, '_'),
	),

	valueLength:  256,
	valuePattern: regexp.MustCompile(`^.*$`),
	valueNormalize: textutil.Chain(
		textutil.ReplaceNotIn(latin1, '_'),
	),
}
