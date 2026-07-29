package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
)

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
			name: "skips ALL_PRIVILEGES removal when granting ALL_PRIVILEGES",
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

func TestNormalizeAssignments(t *testing.T) {
	tests := []struct {
		name     string
		input    []catalog.PrivilegeAssignment
		expected []catalog.PrivilegeAssignment
	}{
		{
			name: "sorts privileges",
			input: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeUseSchema, catalog.PrivilegeApplyTag}},
			},
			expected: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag, catalog.PrivilegeUseSchema}},
			},
		},
		{
			// Regression test for #6030: ALL_PRIVILEGES implies every concrete
			// privilege, so a principal holding it collapses to just
			// ALL_PRIVILEGES. Applied to both config and remote, this stops the
			// backend's extra concrete privileges from showing as drift.
			name: "collapses ALL_PRIVILEGES with concrete privileges",
			input: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeUseCatalog, catalog.PrivilegeAllPrivileges, catalog.PrivilegeCreateSchema}},
			},
			expected: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeAllPrivileges}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizeAssignments(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}
