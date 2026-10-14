package u2m

import (
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
)

func testParams() TokenRequestParams {
	return TokenRequestParams{
		ClientID:  "client",
		Scopes:    []string{"all-apis", "sql"},
		Resources: []string{testHost + "/a", testHost + "/b"},
	}
}

func TestChecksumIncludesEachSetting(t *testing.T) {
	for _, tt := range []struct {
		name   string
		params TokenRequestParams
	}{
		{
			name: "client ID",
			params: TokenRequestParams{
				ClientID:  "other",
				Scopes:    []string{"all-apis", "sql"},
				Resources: []string{testHost + "/a", testHost + "/b"},
			},
		},
		{
			name: "scopes",
			params: TokenRequestParams{
				ClientID:  "client",
				Scopes:    []string{"sql"},
				Resources: []string{testHost + "/a", testHost + "/b"},
			},
		},
		{
			name: "resources",
			params: TokenRequestParams{
				ClientID:  "client",
				Scopes:    []string{"all-apis", "sql"},
				Resources: []string{testHost + "/other"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEqual(t, checksum(testParams()), checksum(tt.params))
		})
	}
}

func TestChecksumEquivalentSettings(t *testing.T) {
	for _, tt := range []struct {
		name   string
		params TokenRequestParams
	}{
		{
			name: "order",
			params: TokenRequestParams{
				ClientID:  "client",
				Scopes:    []string{"sql", "all-apis"},
				Resources: []string{testHost + "/b", testHost + "/a"},
			},
		},
		{
			name: "duplicates and blanks",
			params: TokenRequestParams{
				ClientID:  "client",
				Scopes:    []string{" sql ", "", "all-apis", "sql"},
				Resources: []string{testHost + "/a", " ", testHost + "/b", testHost + "/a"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, checksum(testParams()), checksum(tt.params))
		})
	}
}

func TestChecksumDefaults(t *testing.T) {
	defaults := TokenRequestParams{
		ClientID: appClientID,
		Scopes:   []string{defaultScope},
	}
	for _, tt := range []struct {
		name   string
		params TokenRequestParams
	}{
		{
			name: "nothing set",
		},
		{
			name: "empty scopes",
			params: TokenRequestParams{
				Scopes: []string{},
			},
		},
		{
			name: "blank scopes",
			params: TokenRequestParams{
				Scopes: []string{" ", ""},
			},
		},
		{
			name: "empty resources",
			params: TokenRequestParams{
				Resources: []string{},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, checksum(defaults), checksum(tt.params))
		})
	}
}

func TestChecksumEncoding(t *testing.T) {
	// Checksums are stored in users' token caches. If this value changes, every
	// user with a cached profile token has to log in again. It is the SHA-256 of:
	// {"client_id":"client","scopes":["all-apis","sql"],
	// "resources":["https://workspace.test/a","https://workspace.test/b"]}
	got := checksum(testParams())

	assert.Equal(t, "60224281adde6246e2c82827b71a8f973f4190ea1de3456a92a34a17b0ea2212", got)
}

func TestTokenRequestParamsFromProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile *profile.Profile
		want    TokenRequestParams
	}{
		{
			name: "U2M profile",
			profile: &profile.Profile{
				AuthType:  AuthTypeDatabricksCLI,
				ClientID:  "custom-client",
				Scopes:    "sql, all-apis",
				Resources: "https://workspace.test/a,https://workspace.test/b",
			},
			want: TokenRequestParams{
				ClientID:  "custom-client",
				Scopes:    []string{"sql", "all-apis"},
				Resources: []string{"https://workspace.test/a", "https://workspace.test/b"},
			},
		},
		{
			// Client ID and resources belong to databricks-cli profiles only.
			name: "profile without auth type",
			profile: &profile.Profile{
				ClientID:  "custom-client",
				Scopes:    "sql",
				Resources: "https://workspace.test/a",
			},
			want: TokenRequestParams{
				Scopes: []string{"sql"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TokenRequestParamsFromProfile(tt.profile))
		})
	}
}
