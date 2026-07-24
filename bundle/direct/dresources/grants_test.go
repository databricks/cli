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
		remote   []catalog.PrivilegeAssignment
		expected []catalog.PermissionsChange
	}{
		{
			name: "adds privileges for a new principal (no remote)",
			desired: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag, catalog.PrivilegeCreateTable}},
			},
			expected: []catalog.PermissionsChange{
				{Principal: "alice", Add: []catalog.Privilege{catalog.PrivilegeApplyTag, catalog.PrivilegeCreateTable}},
			},
		},
		{
			name: "removes privileges present in remote but not desired",
			desired: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag}},
			},
			remote: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag, catalog.PrivilegeCreateTable}},
			},
			expected: []catalog.PermissionsChange{
				{Principal: "alice", Remove: []catalog.Privilege{catalog.PrivilegeCreateTable}},
			},
		},
		{
			name: "revokes a principal absent from desired",
			desired: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag}},
			},
			remote: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag}},
				{Principal: "bob", Privileges: []catalog.Privilege{catalog.PrivilegeCreateTable}},
			},
			expected: []catalog.PermissionsChange{
				{Principal: "bob", Remove: []catalog.Privilege{catalog.PrivilegeCreateTable}},
			},
		},
		{
			// Regression test for #6030: a principal granted ALL_PRIVILEGES whose
			// remote also has concrete privileges must have those removed so the
			// deploy converges, instead of leaving them in place forever.
			name: "removes concrete privileges when desired is ALL_PRIVILEGES",
			desired: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeAllPrivileges}},
			},
			remote: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeAllPrivileges, catalog.PrivilegeUseCatalog}},
			},
			expected: []catalog.PermissionsChange{
				{Principal: "alice", Remove: []catalog.Privilege{catalog.PrivilegeUseCatalog}},
			},
		},
		{
			name: "no change when desired equals remote",
			desired: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeAllPrivileges}},
			},
			remote: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeAllPrivileges}},
			},
			expected: []catalog.PermissionsChange{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildGrantChanges(tt.desired, tt.remote))
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
			name: "uppercases and sorts privileges",
			input: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{"use_schema", "apply_tag"}},
			},
			expected: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{"APPLY_TAG", "USE_SCHEMA"}},
			},
		},
		{
			name: "converts spaces to underscores",
			input: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{"create table", "USE SCHEMA"}},
			},
			expected: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{"CREATE_TABLE", "USE_SCHEMA"}},
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
