package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
)

func TestRemovedPrincipalChanges(t *testing.T) {
	tests := []struct {
		name     string
		current  []catalog.PrivilegeAssignment
		desired  []catalog.PrivilegeAssignment
		expected []catalog.PermissionsChange
	}{
		{
			name: "removes principals missing from config",
			current: []catalog.PrivilegeAssignment{
				{
					Principal:  "alice",
					Privileges: []catalog.Privilege{catalog.PrivilegeCreateTable},
				},
				{
					Principal:  "bob",
					Privileges: []catalog.Privilege{catalog.PrivilegeUseSchema, catalog.PrivilegeApplyTag},
				},
			},
			desired: []catalog.PrivilegeAssignment{
				{
					Principal:  "alice",
					Privileges: []catalog.Privilege{catalog.PrivilegeCreateTable},
				},
			},
			expected: []catalog.PermissionsChange{
				{
					Principal: "bob",
					Remove: []catalog.Privilege{
						catalog.PrivilegeApplyTag,
						catalog.PrivilegeUseSchema,
					},
				},
			},
		},
		{
			name: "skips principals that are still desired",
			current: []catalog.PrivilegeAssignment{
				{
					Principal:  "alice",
					Privileges: []catalog.Privilege{catalog.PrivilegeCreateTable},
				},
			},
			desired: []catalog.PrivilegeAssignment{
				{
					Principal:  "alice",
					Privileges: []catalog.Privilege{catalog.PrivilegeCreateTable},
				},
			},
		},
		{
			name: "sorts removed principals and privileges",
			current: []catalog.PrivilegeAssignment{
				{
					Principal:  "charlie",
					Privileges: []catalog.Privilege{catalog.PrivilegeModify},
				},
				{
					Principal:  "bob",
					Privileges: []catalog.Privilege{catalog.PrivilegeUseSchema, catalog.PrivilegeApplyTag},
				},
				{
					Principal: "empty",
				},
			},
			desired: nil,
			expected: []catalog.PermissionsChange{
				{
					Principal: "bob",
					Remove: []catalog.Privilege{
						catalog.PrivilegeApplyTag,
						catalog.PrivilegeUseSchema,
					},
				},
				{
					Principal: "charlie",
					Remove: []catalog.Privilege{
						catalog.PrivilegeModify,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, removedPrincipalChanges(tt.current, tt.desired))
		})
	}
}
