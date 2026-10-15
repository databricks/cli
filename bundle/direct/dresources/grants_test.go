package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"

	grants "github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGrantChanges(t *testing.T) {
	tests := []struct {
		name     string
		desired  []grants.PrivilegeAssignment
		remote   map[string][]string
		expected []grants.PermissionsChange
	}{
		{
			name: "removes all other privileges for desired principal",
			desired: []grants.PrivilegeAssignment{
				{
					Principal: new("alice"),
					Privileges: []string{
						"APPLY_TAG",
						"CREATE_TABLE",
					},
				},
			},
			expected: []grants.PermissionsChange{
				{
					Principal: new("alice"),
					Add: []string{
						"APPLY_TAG",
						"CREATE_TABLE",
					},
					Remove: []string{
						"ALL_PRIVILEGES",
					},
				},
			},
		},
		{
			name: "skips ALL_PRIVILEGES removal when granting ALL_PRIVILEGES",
			desired: []grants.PrivilegeAssignment{
				{
					Principal: new("alice"),
					Privileges: []string{
						"ALL_PRIVILEGES",
					},
				},
			},
			remote: map[string][]string{
				"bob": {"ALL_PRIVILEGES"},
			},
			expected: []grants.PermissionsChange{
				{
					Principal: new("alice"),
					Add: []string{
						"ALL_PRIVILEGES",
					},
				},
				{
					Principal: new("bob"),
					Remove: []string{
						"ALL_PRIVILEGES",
					},
				},
			},
		},
		{
			// ALL_PRIVILEGES stays granted, so MANAGE (which it does not imply and
			// which its removal would not clear) must be revoked by name.
			name: "revokes excluded privilege dropped while keeping ALL_PRIVILEGES",
			desired: []grants.PrivilegeAssignment{
				{
					Principal:  new("alice"),
					Privileges: []string{"ALL_PRIVILEGES"},
				},
			},
			remote: map[string][]string{
				"alice": {"ALL_PRIVILEGES", "MANAGE"},
			},
			expected: []grants.PermissionsChange{
				{
					Principal: new("alice"),
					Add:       []string{"ALL_PRIVILEGES"},
					Remove:    []string{"MANAGE"},
				},
			},
		},
		{
			// Removing a principal that holds MANAGE: ALL_PRIVILEGES removal clears
			// everything it implies, and MANAGE (which it does not imply) is revoked
			// by name so the principal is fully removed instead of lingering.
			name: "revokes excluded privilege of a removed principal",
			remote: map[string][]string{
				"alice": {"ALL_PRIVILEGES", "MANAGE"},
			},
			expected: []grants.PermissionsChange{
				{
					Principal: new("alice"),
					Remove:    []string{"ALL_PRIVILEGES", "MANAGE"},
				},
			},
		},
		{
			// Downgrading a principal off ALL_PRIVILEGES: MANAGE must be revoked by
			// name too, since removing ALL_PRIVILEGES does not clear it.
			name: "revokes excluded privilege when downgrading off ALL_PRIVILEGES",
			desired: []grants.PrivilegeAssignment{
				{
					Principal:  new("alice"),
					Privileges: []string{"SELECT"},
				},
			},
			remote: map[string][]string{
				"alice": {"ALL_PRIVILEGES", "MANAGE"},
			},
			expected: []grants.PermissionsChange{
				{
					Principal: new("alice"),
					Add:       []string{"SELECT"},
					Remove:    []string{"ALL_PRIVILEGES", "MANAGE"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildGrantChanges(tt.desired, tt.remote))
		})
	}
}

// Calls through the adapter so the optional-method wiring is covered too.
func TestGrantsIsEmptyState(t *testing.T) {
	tests := []struct {
		name     string
		state    *GrantsState
		expected bool
	}{
		{
			name:     "empty grants list",
			state:    &GrantsState{SecurableType: "schema", EmbeddedSlice: []grants.PrivilegeAssignment{}},
			expected: true,
		},
		{
			name:     "unset grants list",
			state:    &GrantsState{SecurableType: "schema"},
			expected: true,
		},
		{
			name: "one assignment",
			state: &GrantsState{
				SecurableType: "schema",
				EmbeddedSlice: []grants.PrivilegeAssignment{
					{Principal: new("alice"), Privileges: []string{"SELECT"}},
				},
			},
			expected: false,
		},
	}

	adapter, err := NewAdapter(SupportedResources["schemas.grants"], "schemas.grants", nil)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			empty, err := adapter.IsEmptyState(tt.state)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, empty)
		})
	}
}

// Resources without IsEmptyState are never treated as empty.
func TestIsEmptyStateNotImplemented(t *testing.T) {
	adapter, err := NewAdapter(SupportedResources["schemas"], "schemas", nil)
	require.NoError(t, err)

	empty, err := adapter.IsEmptyState(&resources.CreateSchema{Name: new("myschema")})
	require.NoError(t, err)
	assert.False(t, empty)
}

func TestNormalizeAssignments(t *testing.T) {
	tests := []struct {
		name     string
		input    []grants.PrivilegeAssignment
		expected []grants.PrivilegeAssignment
	}{
		{
			name: "sorts privileges",
			input: []grants.PrivilegeAssignment{
				{Principal: new("alice"), Privileges: []string{"USE_SCHEMA", "APPLY_TAG"}},
			},
			expected: []grants.PrivilegeAssignment{
				{Principal: new("alice"), Privileges: []string{"APPLY_TAG", "USE_SCHEMA"}},
			},
		},
		{
			// Regression test for #6030: ALL_PRIVILEGES implies every concrete
			// privilege, so a principal holding it collapses to just
			// ALL_PRIVILEGES. Applied to both config and remote, this stops the
			// backend's extra concrete privileges from showing as drift.
			name: "collapses ALL_PRIVILEGES with concrete privileges",
			input: []grants.PrivilegeAssignment{
				{Principal: new("alice"), Privileges: []string{"USE_CATALOG", "ALL_PRIVILEGES", "CREATE_SCHEMA"}},
			},
			expected: []grants.PrivilegeAssignment{
				{Principal: new("alice"), Privileges: []string{"ALL_PRIVILEGES"}},
			},
		},
		{
			// ALL_PRIVILEGES does not imply MANAGE, READ_METADATA, EXTERNAL_USE_SCHEMA
			// or EXTERNAL_USE_LOCATION, so all four must survive the collapse instead
			// of being dropped alongside the concrete privileges (e.g. USE_CATALOG).
			name: "keeps privileges not implied by ALL_PRIVILEGES",
			input: []grants.PrivilegeAssignment{
				{Principal: new("alice"), Privileges: []string{
					"USE_CATALOG",
					"MANAGE",
					"READ_METADATA",
					"EXTERNAL_USE_SCHEMA",
					"EXTERNAL_USE_LOCATION",
					"ALL_PRIVILEGES",
				}},
			},
			expected: []grants.PrivilegeAssignment{
				{Principal: new("alice"), Privileges: []string{
					"ALL_PRIVILEGES",
					"EXTERNAL_USE_LOCATION",
					"EXTERNAL_USE_SCHEMA",
					"MANAGE",
					"READ_METADATA",
				}},
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
