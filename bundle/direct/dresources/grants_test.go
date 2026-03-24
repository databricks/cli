package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemovedGrantPrincipals(t *testing.T) {
	tests := []struct {
		name     string
		changes  Changes
		desired  []catalog.PrivilegeAssignment
		expected []string
	}{
		{
			name: "finds removed principals from root keyed path",
			changes: Changes{
				"[principal='bob']": {
					Remote: catalog.PrivilegeAssignment{
						Principal: "bob",
					},
				},
			},
			desired: []catalog.PrivilegeAssignment{
				{
					Principal: "alice",
				},
			},
			expected: []string{
				"bob",
			},
		},
		{
			name: "finds removed principals from nested privilege paths",
			changes: Changes{
				"[principal='bob'].privileges[0]": {
					Remote: catalog.PrivilegeUseSchema,
				},
				"[principal='alice'].privileges[0]": {
					Remote: catalog.PrivilegeCreateTable,
				},
			},
			desired: []catalog.PrivilegeAssignment{
				{
					Principal: "alice",
				},
			},
			expected: []string{
				"bob",
			},
		},
		{
			name: "skips desired principals and nil remote entries",
			changes: Changes{
				"[principal='alice'].privileges[0]": {
					Remote: catalog.PrivilegeCreateTable,
				},
				"[principal='bob'].privileges[0]": {},
			},
			desired: []catalog.PrivilegeAssignment{
				{
					Principal: "alice",
				},
			},
		},
		{
			name: "sorts removed principals and unescapes quotes",
			changes: Changes{
				"[principal='team''s group'].privileges[0]": {
					Remote: catalog.PrivilegeUseSchema,
				},
				"[principal='bob'].privileges[0]": {
					Remote: catalog.PrivilegeApplyTag,
				},
			},
			expected: []string{
				"bob",
				"team's group",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := removedGrantPrincipals(tt.changes, tt.desired)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestBuildGrantChanges(t *testing.T) {
	tests := []struct {
		name     string
		desired  []catalog.PrivilegeAssignment
		removed  []string
		expected []catalog.PermissionsChange
	}{
		{
			name: "removes all other privileges for desired principal",
			desired: []catalog.PrivilegeAssignment{
				{
					Principal: "alice",
					Privileges: []catalog.Privilege{
						catalog.PrivilegeApplyTag,
						catalog.PrivilegeCreateTable,
					},
				},
			},
			expected: []catalog.PermissionsChange{
				{
					Principal: "alice",
					Add: []catalog.Privilege{
						catalog.PrivilegeApplyTag,
						catalog.PrivilegeCreateTable,
					},
					Remove: []catalog.Privilege{
						catalog.PrivilegeAllPrivileges,
					},
				},
			},
		},
		{
			name: "uses all privileges for removed principals",
			desired: []catalog.PrivilegeAssignment{
				{
					Principal: "alice",
					Privileges: []catalog.Privilege{
						catalog.PrivilegeAllPrivileges,
					},
				},
			},
			removed: []string{
				"bob",
			},
			expected: []catalog.PermissionsChange{
				{
					Principal: "alice",
					Add: []catalog.Privilege{
						catalog.PrivilegeAllPrivileges,
					},
				},
				{
					Principal: "bob",
					Remove: []catalog.Privilege{
						catalog.PrivilegeAllPrivileges,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildGrantChanges(tt.desired, tt.removed))
		})
	}
}
