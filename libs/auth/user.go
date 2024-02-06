package auth

import (
	"strings"

	"github.com/databricks/cli/libs/textutil"
)

// Get a short-form username, based on the user's primary email address.
// We leave the full range of unicode letters in tact, but remove all "special" characters,
// including dots, which are not supported in e.g. experiment names.
func GetShortUserName(emailAddress string) string {
	local, _, _ := strings.Cut(emailAddress, "@")
	return textutil.NormalizeString(local)
}
