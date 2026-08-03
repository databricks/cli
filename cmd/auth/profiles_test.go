package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileMetadataIsEmpty(t *testing.T) {
	cases := []struct {
		name  string
		meta  profileMetadata
		empty bool
	}{
		{"no host or account", profileMetadata{Name: "p"}, true},
		{"host set", profileMetadata{Name: "p", Host: "https://x.cloud.databricks.test"}, false},
		{"account set", profileMetadata{Name: "p", AccountID: "acct"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.empty, tc.meta.IsEmpty())
		})
	}
}

func TestCloudFromHost(t *testing.T) {
	cases := []struct {
		host  string
		cloud string
	}{
		{"https://abc.cloud.databricks.com", "aws"},
		{"https://adb-123.4.azuredatabricks.net", "azure"},
		{"https://123.4.gcp.databricks.com", "gcp"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			assert.Equal(t, tc.cloud, cloudFromHost(tc.host))
		})
	}
}

func TestAuthTypeFromKeys(t *testing.T) {
	const host = "https://abc.cloud.databricks.test"
	cases := []struct {
		name     string
		keys     map[string]string
		authType string
	}{
		{
			// databricks auth login writes auth_type explicitly.
			"explicit auth_type wins over inference",
			map[string]string{"host": host, "token": "dapi", "auth_type": "databricks-cli"},
			"databricks-cli",
		},
		{"explicit pat", map[string]string{"host": host, "token": "dapi", "auth_type": "pat"}, "pat"},
		{"token implies pat", map[string]string{"host": host, "token": "dapi"}, "pat"},
		{"username and password imply basic", map[string]string{"host": host, "username": "u", "password": "p"}, "basic"},
		// PAT and basic both need a host, matching the SDK's strategies.
		{"token without host", map[string]string{"token": "dapi"}, ""},
		{"password without username", map[string]string{"host": host, "password": "p"}, ""},
		// oauth-m2m needs an OIDC round trip to confirm, so it stays empty here
		// just as it did when the command validated.
		{"client credentials stay empty", map[string]string{"host": host, "client_id": "c", "client_secret": "s"}, ""},
		{"host only", map[string]string{"host": host}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.authType, authTypeFromKeys(tc.keys))
		})
	}
}

// The flag is a no-op kept for callers that still pass it (e.g. the VS Code
// extension); it must parse without error and change nothing.
func TestProfilesSkipValidateFlagIsAcceptedAndHidden(t *testing.T) {
	cmd := newProfilesCommand()
	flag := cmd.Flags().Lookup("skip-validate")
	if assert.NotNil(t, flag, "--skip-validate must still be accepted") {
		assert.True(t, flag.Hidden, "--skip-validate should be hidden")
	}
	require.NoError(t, cmd.Flags().Parse([]string{"--skip-validate"}))
}
