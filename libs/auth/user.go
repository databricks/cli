package auth

import (
	"strings"

	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// Get a short-form username, based on the user's primary email address.
// We leave the full range of unicode letters in tact, but remove all "special" characters,
// including dots, which are not supported in e.g. experiment names.
func GetShortUserName(user *iam.User) string {
	var name string
	if IsServicePrincipal(user.UserName) {
		name = user.DisplayName
	}
	if name == "" {
		name = user.UserName
	}
	local, _, _ := strings.Cut(name, "@")
	return textutil.NormalizeString(local)
}
