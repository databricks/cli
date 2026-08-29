package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsWorkspaceProxyPath(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "assignable roles proxy",
			path: "/api/2.0/preview/accounts/access-control/assignable-roles",
			want: true,
		},
		{
			name: "rule sets proxy",
			path: "/api/2.0/preview/accounts/access-control/rule-sets",
			want: true,
		},
		{
			name: "service principal secrets proxy",
			path: "/api/2.0/accounts/servicePrincipals/spn-123/credentials/secrets",
			want: true,
		},
		{
			name: "account service principal secrets path has account id segment",
			path: "/api/2.0/accounts/abc-123/servicePrincipals/spn-123/credentials/secrets",
			want: false,
		},
		{
			name: "rule sets child is not part of exact proxy entry",
			path: "/api/2.0/preview/accounts/access-control/rule-sets/foo",
			want: false,
		},
		{
			name: "workspace path",
			path: "/api/2.0/clusters/list",
			want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, isWorkspaceProxyPath(c.path))
		})
	}
}
