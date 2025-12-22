package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func TestCollapsePermissions(t *testing.T) {
	tests := []struct {
		name        string
		input       []resources.SecretScopePermission
		expected    []resources.SecretScopePermission
		expectError bool
	}{
		{
			name: "no duplicates",
			input: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
				{UserName: "user2", Level: resources.SecretScopePermissionLevelWrite},
			},
			expected: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
				{UserName: "user2", Level: resources.SecretScopePermissionLevelWrite},
			},
			expectError: false,
		},
		{
			name: "duplicate user - keep highest (MANAGE over WRITE)",
			input: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelWrite},
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
			},
			expected: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
			},
			expectError: false,
		},
		{
			name: "duplicate user - keep highest (MANAGE over READ)",
			input: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelRead},
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
			},
			expected: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
			},
			expectError: false,
		},
		{
			name: "duplicate user - keep highest (WRITE over READ)",
			input: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelRead},
				{UserName: "user1", Level: resources.SecretScopePermissionLevelWrite},
			},
			expected: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelWrite},
			},
			expectError: false,
		},
		{
			name: "duplicate group - keep highest",
			input: []resources.SecretScopePermission{
				{GroupName: "group1", Level: resources.SecretScopePermissionLevelRead},
				{GroupName: "group1", Level: resources.SecretScopePermissionLevelManage},
			},
			expected: []resources.SecretScopePermission{
				{GroupName: "group1", Level: resources.SecretScopePermissionLevelManage},
			},
			expectError: false,
		},
		{
			name: "duplicate service principal - keep highest",
			input: []resources.SecretScopePermission{
				{ServicePrincipalName: "sp1", Level: resources.SecretScopePermissionLevelWrite},
				{ServicePrincipalName: "sp1", Level: resources.SecretScopePermissionLevelManage},
			},
			expected: []resources.SecretScopePermission{
				{ServicePrincipalName: "sp1", Level: resources.SecretScopePermissionLevelManage},
			},
			expectError: false,
		},
		{
			name: "multiple duplicates across different principal types",
			input: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelRead},
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
				{GroupName: "group1", Level: resources.SecretScopePermissionLevelWrite},
				{GroupName: "group1", Level: resources.SecretScopePermissionLevelRead},
				{ServicePrincipalName: "sp1", Level: resources.SecretScopePermissionLevelRead},
				{ServicePrincipalName: "sp1", Level: resources.SecretScopePermissionLevelWrite},
			},
			expected: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
				{GroupName: "group1", Level: resources.SecretScopePermissionLevelWrite},
				{ServicePrincipalName: "sp1", Level: resources.SecretScopePermissionLevelWrite},
			},
			expectError: false,
		},
		{
			name: "three duplicates - keep highest",
			input: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelRead},
				{UserName: "user1", Level: resources.SecretScopePermissionLevelWrite},
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
			},
			expected: []resources.SecretScopePermission{
				{UserName: "user1", Level: resources.SecretScopePermissionLevelManage},
			},
			expectError: false,
		},
		{
			name:        "empty permissions",
			input:       []resources.SecretScopePermission{},
			expected:    []resources.SecretScopePermission{},
			expectError: false,
		},
		{
			name: "unknown permission level",
			input: []resources.SecretScopePermission{
				{UserName: "user1", Level: "UNKNOWN"},
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := &resources.SecretScope{
				Permissions: tt.input,
			}

			err := collapsePermissions(scope)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expected, scope.Permissions)
			}
		})
	}
}

func TestAddManageForCurrentUser(t *testing.T) {
	tests := []struct {
		name             string
		currentUser      *iam.User
		existingPerms    []resources.SecretScopePermission
		expectedPerms    []resources.SecretScopePermission
		shouldAddNewPerm bool
	}{
		{
			name:          "add MANAGE for regular user when no permissions exist",
			currentUser:   &iam.User{UserName: "user@example.com"},
			existingPerms: []resources.SecretScopePermission{},
			expectedPerms: []resources.SecretScopePermission{
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelManage},
			},
			shouldAddNewPerm: true,
		},
		{
			name:          "add MANAGE for service principal when no permissions exist",
			currentUser:   &iam.User{UserName: "12345678-1234-1234-1234-123456789012"},
			existingPerms: []resources.SecretScopePermission{},
			expectedPerms: []resources.SecretScopePermission{
				{ServicePrincipalName: "12345678-1234-1234-1234-123456789012", Level: resources.SecretScopePermissionLevelManage},
			},
			shouldAddNewPerm: true,
		},
		{
			name:        "do not add if user already has MANAGE",
			currentUser: &iam.User{UserName: "user@example.com"},
			existingPerms: []resources.SecretScopePermission{
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelManage},
			},
			expectedPerms: []resources.SecretScopePermission{
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelManage},
			},
			shouldAddNewPerm: false,
		},
		{
			name:        "add MANAGE if user has READ",
			currentUser: &iam.User{UserName: "user@example.com"},
			existingPerms: []resources.SecretScopePermission{
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelRead},
			},
			expectedPerms: []resources.SecretScopePermission{
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelRead},
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelManage},
			},
			shouldAddNewPerm: true,
		},
		{
			name:        "add MANAGE if user has WRITE",
			currentUser: &iam.User{UserName: "user@example.com"},
			existingPerms: []resources.SecretScopePermission{
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelWrite},
			},
			expectedPerms: []resources.SecretScopePermission{
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelWrite},
				{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelManage},
			},
			shouldAddNewPerm: true,
		},
		{
			name:        "do not add if service principal already has MANAGE",
			currentUser: &iam.User{UserName: "12345678-1234-1234-1234-123456789012"},
			existingPerms: []resources.SecretScopePermission{
				{ServicePrincipalName: "12345678-1234-1234-1234-123456789012", Level: resources.SecretScopePermissionLevelManage},
			},
			expectedPerms: []resources.SecretScopePermission{
				{ServicePrincipalName: "12345678-1234-1234-1234-123456789012", Level: resources.SecretScopePermissionLevelManage},
			},
			shouldAddNewPerm: false,
		},
		{
			name:        "add MANAGE when other users have permissions",
			currentUser: &iam.User{UserName: "user1@example.com"},
			existingPerms: []resources.SecretScopePermission{
				{UserName: "user2@example.com", Level: resources.SecretScopePermissionLevelRead},
				{GroupName: "group1", Level: resources.SecretScopePermissionLevelWrite},
			},
			expectedPerms: []resources.SecretScopePermission{
				{UserName: "user2@example.com", Level: resources.SecretScopePermissionLevelRead},
				{GroupName: "group1", Level: resources.SecretScopePermissionLevelWrite},
				{UserName: "user1@example.com", Level: resources.SecretScopePermissionLevelManage},
			},
			shouldAddNewPerm: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := &resources.SecretScope{
				Permissions: tt.existingPerms,
			}

			initialCount := len(scope.Permissions)
			addManageForCurrentUser(scope, tt.currentUser)

			assert.ElementsMatch(t, tt.expectedPerms, scope.Permissions)

			if tt.shouldAddNewPerm {
				assert.Equal(t, initialCount+1, len(scope.Permissions))
			} else {
				assert.Equal(t, initialCount, len(scope.Permissions))
			}
		})
	}
}
