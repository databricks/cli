package u2m

import (
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/textutil"
)

// AuthTypeDatabricksCLI is the auth_type of profiles that use the CLI's U2M
// login. It is unrelated to appClientID, the OAuth client ID with the same
// text.
const AuthTypeDatabricksCLI = "databricks-cli"

// ClientIDFromProfile excludes client IDs belonging to other auth types.
func ClientIDFromProfile(p *profile.Profile) string {
	if p == nil || p.AuthType != AuthTypeDatabricksCLI {
		return ""
	}
	return p.ClientID
}

// ResourcesFromProfile returns the RFC 8707 resource indicators saved on a
// databricks-cli-auth profile, so `login`/`auth token` re-request the same ones.
func ResourcesFromProfile(p *profile.Profile) []string {
	if p == nil || p.AuthType != AuthTypeDatabricksCLI {
		return nil
	}
	return textutil.SplitAndTrim(p.Resources)
}
