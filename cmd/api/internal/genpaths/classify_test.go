package main

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    classification
		wantErr string
	}{
		{
			name: "literal proxy path -> exact match",
			src:  `"/api/2.0/preview/accounts/access-control/rule-sets"`,
			want: classification{class: classWorkspaceProxyExact, value: "/api/2.0/preview/accounts/access-control/rule-sets"},
		},
		{
			name: "literal path without accounts segment is ignored",
			src:  `"/api/2.0/clusters/list"`,
			want: classification{class: classNotAccount},
		},
		{
			name: "Sprintf without account-ID arg -> prefix",
			src:  `fmt.Sprintf("/api/2.0/accounts/servicePrincipals/%v/credentials/secrets", request.ServicePrincipalId)`,
			want: classification{class: classWorkspaceProxyPrefix, value: "/api/2.0/accounts/servicePrincipals/"},
		},
		{
			name: "Sprintf with ConfiguredAccountID -> account",
			src:  `fmt.Sprintf("/api/2.0/accounts/%v/scim/v2/Groups", a.client.ConfiguredAccountID())`,
			want: classification{class: classAccountAPI},
		},
		{
			name: "Sprintf with cfg.AccountID -> account",
			src:  `fmt.Sprintf("/api/2.0/accounts/%v/foo", cfg.AccountID)`,
			want: classification{class: classAccountAPI},
		},
		{
			name: "Sprintf with a.client.Config.AccountID -> account",
			src:  `fmt.Sprintf("/api/2.0/accounts/%v/foo", a.client.Config.AccountID)`,
			want: classification{class: classAccountAPI},
		},
		{
			name: "Sprintf multi-arg with ConfiguredAccountID -> account",
			src:  `fmt.Sprintf("/api/2.0/accounts/%v/workspaces/%v/permissionassignments", a.client.ConfiguredAccountID(), request.WorkspaceId)`,
			want: classification{class: classAccountAPI},
		},
		{
			name: "Sprintf without accounts segment is ignored",
			src:  `fmt.Sprintf("/api/2.0/clusters/%v/events", request.ClusterID)`,
			want: classification{class: classNotAccount},
		},
		{
			name:    "string concatenation that mentions accounts/ -> error",
			src:     `"/api/2.0/preview/accounts/" + suffix`,
			wantErr: "unrecognized construction idiom",
		},
		{
			name:    "helper function call that mentions accounts/ -> error",
			src:     `buildPath("/api/2.0/preview/accounts/foo")`,
			wantErr: "unrecognized construction idiom",
		},
		{
			name:    "Sprintf with unrecognized account-ID source -> error (guard 1)",
			src:     `fmt.Sprintf("/api/foo/accounts/%v/bar", request.AccountId)`,
			wantErr: `format verb immediately after "accounts/"`,
		},
		{
			name:    "Sprintf with locally-aliased ConfiguredAccountID -> error (guard 1)",
			src:     `fmt.Sprintf("/api/2.0/accounts/%v/scim/v2/Groups", accID)`,
			wantErr: `format verb immediately after "accounts/"`,
		},
		{
			name:    "Sprintf with verb before accounts segment -> error",
			src:     `fmt.Sprintf("/api/2.0/%v/accounts/servicePrincipals/%v/credentials/secrets", request.Scope, request.ServicePrincipalId)`,
			wantErr: `format verb before the "accounts/" segment`,
		},
		{
			name:    "Sprintf with non-client receiver on ConfiguredAccountID -> error",
			src:     `fmt.Sprintf("/api/2.0/accounts/%v/foo", request.ConfiguredAccountID())`,
			wantErr: `format verb immediately after "accounts/"`,
		},
		{
			name: "Sprintf with %% literal before real verb still classifies correctly",
			src:  `fmt.Sprintf("/api/2.0/accounts/servicePrincipals/%%marker/%v/credentials/secrets", request.ServicePrincipalId)`,
			want: classification{class: classWorkspaceProxyPrefix, value: "/api/2.0/accounts/servicePrincipals/%marker/"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(c.src)
			require.NoError(t, err, "fixture must parse")
			got, gotErr := classify(expr)
			if c.wantErr != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), c.wantErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, c.want, got)
		})
	}
}

// TestScanSDK runs the generator end-to-end against the pinned SDK. The
// expected sets are exact: a regression that adds an entry (overbroad prefix,
// new shape misclassified) is just as bad as one that drops an entry.
//
// When bumping the SDK pin, expected diffs go through `./task generate-paths`.
// Update this test only after confirming the new entries are intentional.
func TestScanSDK(t *testing.T) {
	dir, err := resolveSDKDir()
	require.NoError(t, err)

	prefixes, exacts, err := scanSDK(dir)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"/api/2.0/accounts/servicePrincipals/",
	}, prefixes)
	assert.Equal(t, []string{
		"/api/2.0/preview/accounts/access-control/assignable-roles",
		"/api/2.0/preview/accounts/access-control/rule-sets",
	}, exacts)
}

// TestScanAST_VarPath verifies that `var path = ...` declarations are scanned
// the same way as `path := ...` assignments. This is a defensive case: the
// pinned SDK doesn't use the `var` form today, but if a future SDK introduces
// it, we want classification rather than silent skip.
func TestScanAST_VarPath(t *testing.T) {
	src := `package svc
type api struct{}
func (a *api) F() {
	var path = "/api/2.0/preview/accounts/access-control/rule-sets"
	_ = path
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	prefixSet := map[string]struct{}{}
	exactSet := map[string]struct{}{}
	require.NoError(t, scanAST(fset, f, prefixSet, exactSet))

	assert.Empty(t, prefixSet)
	assert.Equal(t, map[string]struct{}{
		"/api/2.0/preview/accounts/access-control/rule-sets": {},
	}, exactSet)
}

// TestScanAST_CompoundAssign verifies that compound assignments to `path`
// (like +=) cause the generator to fail loudly. State across statements is
// outside the per-expression classifier's contract.
func TestScanAST_CompoundAssign(t *testing.T) {
	src := `package svc
type api struct{}
func (a *api) F() {
	path := "/api/2.0"
	path += "/preview/accounts/access-control/rule-sets"
	_ = path
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	prefixSet := map[string]struct{}{}
	exactSet := map[string]struct{}{}
	err = scanAST(fset, f, prefixSet, exactSet)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compound assignment")
}
