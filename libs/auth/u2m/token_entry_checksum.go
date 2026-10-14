package u2m

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/textutil"
	"golang.org/x/oauth2"
)

// defaultScope matches the scope resolveScopes requests when none is set.
const defaultScope = "all-apis"

// TokenRequestParams holds the login settings that decide what a token is
// allowed to do. They are sent only when logging in, and the refresh token
// keeps them, so changing them later without a new login would silently
// reuse a token with other permissions.
//
// Settings that decide where requests go, such as the host or the account ID,
// are deliberately excluded. A token used with the wrong host or account is
// rejected by the server, so it is not an access problem.
//
// New fields must use omitempty, so logins that do not set them keep their
// checksum and users do not have to log in again.
type TokenRequestParams struct {
	ClientID  string   `json:"client_id"`
	Scopes    []string `json:"scopes"`
	Resources []string `json:"resources"`
}

// TokenRequestParamsFromProfile returns the login settings saved on a profile,
// so `auth token` and its inline login use the ones the profile was logged in
// with. A missing profile returns the defaults. CLICredentials.tokenParamsFromConfig
// in libs/auth applies the same auth_type rule for other commands.
func TokenRequestParamsFromProfile(p *profile.Profile) TokenRequestParams {
	if p == nil {
		return TokenRequestParams{}
	}
	return TokenRequestParams{
		ClientID:  ClientIDFromProfile(p),
		Scopes:    textutil.SplitAndTrim(p.Scopes),
		Resources: ResourcesFromProfile(p),
	}
}

// NewEntry pairs a token from Challenge with the checksum of the login
// settings that produced it.
func NewEntry(token *oauth2.Token, params TokenRequestParams) storage.Entry {
	return storage.Entry{Token: token, ProfileChecksum: checksum(params)}
}

// checksum hashes normalized params, so equivalent settings match.
//
// The result is stored in users' token caches. Changing the JSON field names,
// their order, or the normalization changes every checksum and makes every
// user log in again.
func checksum(params TokenRequestParams) string {
	// The value contains only strings and string slices, so encoding cannot fail.
	data, _ := json.Marshal(params.normalized())
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// normalized returns a copy with the defaults the OAuth request uses filled in
// and with scopes and resources treated as sets.
func (p TokenRequestParams) normalized() TokenRequestParams {
	if p.ClientID == "" {
		p.ClientID = appClientID
	}
	p.Scopes = normalizeSet(p.Scopes)
	if len(p.Scopes) == 0 {
		p.Scopes = []string{defaultScope}
	}
	p.Resources = normalizeSet(p.Resources)
	return p
}

// normalizeSet treats scopes and resources as sets, so order, duplicates and
// blank entries do not change the checksum. Empty input returns nil, so nil
// and empty slices encode the same way.
func normalizeSet(values []string) []string {
	var normalized []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			normalized = append(normalized, v)
		}
	}
	slices.Sort(normalized)
	return slices.Compact(normalized)
}
