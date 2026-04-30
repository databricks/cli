package api

import (
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasAccountSegment(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"account UUID", "/api/2.0/accounts/123e4567-e89b-12d3-a456-426614174000/ip-access-lists", true},
		{"AIP resource-name shape", "/api/networking/v1/accounts/123e4567-e89b-12d3-a456-426614174000/endpoints/abc", true},
		{"iamv2 account API", "/api/2.0/identity/accounts/123e4567-e89b-12d3-a456-426614174000/workspaces/123/workspaceAccessDetails/abc", true},
		{"non-UUID account ID", "/api/2.0/accounts/abc/foo", true},
		{"hyphenated short ID", "/api/2.0/accounts/abc-123/network-policies", true},
		{"substituted any-shape ID", "/api/2.0/accounts/some-account/oauth2/published-app-integrations", true},

		{"deny-listed exact rule-sets", "/api/2.0/preview/accounts/access-control/rule-sets", false},
		{"deny-listed exact assignable-roles", "/api/2.0/preview/accounts/access-control/assignable-roles", false},
		{"exact-list miss falls to regex (rule-sets/foo)", "/api/2.0/preview/accounts/access-control/rule-sets/foo", true},
		{"exact-list miss falls to regex (assignable-roles-extra)", "/api/2.0/preview/accounts/access-control/assignable-roles-extra", true},
		{"deny-listed prefix servicePrincipals", "/api/2.0/accounts/servicePrincipals/abc-123/credentials/secrets", false},

		{"no accounts segment", "/api/2.0/clusters/list", false},
		{"segment ends in accounts (boundary)", "/api/2.0/some-accounts/abc/foo", false},

		{"query string preserved on match", "/api/2.0/accounts/abc-123?include=foo", true},
		{"query string with /accounts/ does not match", "/api/2.0/clusters/list?next=/accounts/foo", false},
		{"fragment with accounts/ does not match", "/api/2.0/clusters/list#accounts/foo", false},

		{"absolute URL, account path", "https://ignored.example.com/api/2.0/accounts/abc/foo?include=x", true},
		{"absolute URL, query-string accounts/ does not match", "https://ignored.example.com/api/2.0/clusters/list?next=/accounts/foo", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := hasAccountSegment(c.path)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestResolveOrgID(t *testing.T) {
	const (
		workspacePath = "/api/2.0/clusters/list"
		accountPath   = "/api/2.0/accounts/abc-123/network-policies"
		proxyPath     = "/api/2.0/preview/accounts/access-control/rule-sets"
		resolvedWSID  = "900800700600"
		flagWSID      = "999"
	)

	cases := []struct {
		name             string
		forceAccount     bool
		workspaceIDFlag  string
		flagSet          bool
		cfgWorkspaceID   string
		path             string
		want             string
		wantErrSubstring string
	}{
		{
			name:           "empty WorkspaceID + workspace path -> no identifier",
			cfgWorkspaceID: "",
			path:           workspacePath,
			want:           "",
		},
		{
			name:           "WorkspaceID set + workspace path -> sends identifier",
			cfgWorkspaceID: resolvedWSID,
			path:           workspacePath,
			want:           resolvedWSID,
		},
		{
			name:           "WorkspaceID set + account path -> no identifier (auto-detect)",
			cfgWorkspaceID: resolvedWSID,
			path:           accountPath,
			want:           "",
		},
		{
			name:           "WorkspaceID set + workspace-routed proxy under accounts/",
			cfgWorkspaceID: resolvedWSID,
			path:           proxyPath,
			want:           resolvedWSID,
		},
		{
			name:           "--account on workspace path",
			forceAccount:   true,
			cfgWorkspaceID: resolvedWSID,
			path:           workspacePath,
			want:           "",
		},
		{
			name:            "--workspace-id overrides resolved value",
			workspaceIDFlag: flagWSID,
			flagSet:         true,
			cfgWorkspaceID:  resolvedWSID,
			path:            workspacePath,
			want:            flagWSID,
		},
		{
			name:            "--workspace-id on account path still overrides",
			workspaceIDFlag: flagWSID,
			flagSet:         true,
			cfgWorkspaceID:  resolvedWSID,
			path:            accountPath,
			want:            flagWSID,
		},
		{
			name:             "--workspace-id empty value -> error",
			workspaceIDFlag:  "",
			flagSet:          true,
			cfgWorkspaceID:   resolvedWSID,
			path:             workspacePath,
			wantErrSubstring: "--workspace-id requires a value",
		},
		{
			name:             "--account and --workspace-id both set -> error",
			forceAccount:     true,
			workspaceIDFlag:  flagWSID,
			flagSet:          true,
			cfgWorkspaceID:   resolvedWSID,
			path:             workspacePath,
			wantErrSubstring: "mutually exclusive",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolveOrgID(c.forceAccount, c.workspaceIDFlag, c.flagSet, c.cfgWorkspaceID, c.path)
			if c.wantErrSubstring != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.wantErrSubstring)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// TestNormalizeWorkspaceID covers the helper that strips the CLI-only
// WorkspaceIDNone sentinel. RunE calls this directly before resolveOrgID, so
// a regression here would surface as the literal "none" being sent on the
// wire.
func TestNormalizeWorkspaceID(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"sentinel stripped to empty", auth.WorkspaceIDNone, ""},
		{"empty passes through", "", ""},
		{"normal value passes through", "900800700600", "900800700600"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, normalizeWorkspaceID(c.in))
		})
	}
}

func TestSubstituteAccountID(t *testing.T) {
	cases := []struct {
		name             string
		path             string
		accountID        string
		profile          string
		want             string
		wantErrSubstring string
	}{
		{
			name:      "placeholder absent leaves path unchanged",
			path:      "/api/2.0/clusters/list",
			accountID: "abc-123",
			profile:   "DEFAULT",
			want:      "/api/2.0/clusters/list",
		},
		{
			name:      "placeholder present + account_id set",
			path:      "/api/2.0/accounts/{account_id}/oauth2/published-app-integrations",
			accountID: "abc-123",
			profile:   "DEFAULT",
			want:      "/api/2.0/accounts/abc-123/oauth2/published-app-integrations",
		},
		{
			name:      "multiple placeholders all replaced",
			path:      "/api/2.0/accounts/{account_id}/workspaces/123/foo?ref=accounts/{account_id}",
			accountID: "abc-123",
			profile:   "DEFAULT",
			want:      "/api/2.0/accounts/abc-123/workspaces/123/foo?ref=accounts/abc-123",
		},
		{
			name:             "placeholder present + account_id empty -> error",
			path:             "/api/2.0/accounts/{account_id}/oauth2/published-app-integrations",
			accountID:        "",
			profile:          "DEFAULT",
			wantErrSubstring: `{account_id}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := substituteAccountID(c.path, c.accountID, c.profile)
			if c.wantErrSubstring != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.wantErrSubstring)
				assert.Contains(t, err.Error(), `profile "`+c.profile+`"`)
				assert.Contains(t, err.Error(), "DATABRICKS_ACCOUNT_ID")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}
