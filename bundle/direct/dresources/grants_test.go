package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGrantChanges(t *testing.T) {
	tests := []struct {
		name     string
		desired  []catalog.PrivilegeAssignment
		remote   map[string][]catalog.Privilege
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
			remote: map[string][]catalog.Privilege{
				"bob": {catalog.PrivilegeAllPrivileges},
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
		{
			// ALL_PRIVILEGES stays granted, so MANAGE (which it does not imply and
			// which its removal would not clear) must be revoked by name.
			name: "revokes excluded privilege dropped while keeping ALL_PRIVILEGES",
			desired: []catalog.PrivilegeAssignment{
				{
					Principal:  "alice",
					Privileges: []catalog.Privilege{catalog.PrivilegeAllPrivileges},
				},
			},
			remote: map[string][]catalog.Privilege{
				"alice": {catalog.PrivilegeAllPrivileges, catalog.PrivilegeManage},
			},
			expected: []catalog.PermissionsChange{
				{
					Principal: "alice",
					Add:       []catalog.Privilege{catalog.PrivilegeAllPrivileges},
					Remove:    []catalog.Privilege{catalog.PrivilegeManage},
				},
			},
		},
		{
			// Removing a principal that holds MANAGE: ALL_PRIVILEGES removal clears
			// everything it implies, and MANAGE (which it does not imply) is revoked
			// by name so the principal is fully removed instead of lingering.
			name: "revokes excluded privilege of a removed principal",
			remote: map[string][]catalog.Privilege{
				"alice": {catalog.PrivilegeAllPrivileges, catalog.PrivilegeManage},
			},
			expected: []catalog.PermissionsChange{
				{
					Principal: "alice",
					Remove:    []catalog.Privilege{catalog.PrivilegeAllPrivileges, catalog.PrivilegeManage},
				},
			},
		},
		{
			// Downgrading a principal off ALL_PRIVILEGES: MANAGE must be revoked by
			// name too, since removing ALL_PRIVILEGES does not clear it.
			name: "revokes excluded privilege when downgrading off ALL_PRIVILEGES",
			desired: []catalog.PrivilegeAssignment{
				{
					Principal:  "alice",
					Privileges: []catalog.Privilege{catalog.PrivilegeSelect},
				},
			},
			remote: map[string][]catalog.Privilege{
				"alice": {catalog.PrivilegeAllPrivileges, catalog.PrivilegeManage},
			},
			expected: []catalog.PermissionsChange{
				{
					Principal: "alice",
					Add:       []catalog.Privilege{catalog.PrivilegeSelect},
					Remove:    []catalog.Privilege{catalog.PrivilegeAllPrivileges, catalog.PrivilegeManage},
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
			state:    &GrantsState{SecurableType: "schema", EmbeddedSlice: []catalog.PrivilegeAssignment{}},
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
				EmbeddedSlice: []catalog.PrivilegeAssignment{
					{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeSelect}},
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

	empty, err := adapter.IsEmptyState(&catalog.CreateSchema{Name: "myschema"})
	require.NoError(t, err)
	assert.False(t, empty)
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
		{
			// ALL_PRIVILEGES does not imply MANAGE, READ_METADATA, EXTERNAL_USE_SCHEMA
			// or EXTERNAL_USE_LOCATION, so all four must survive the collapse instead
			// of being dropped alongside the concrete privileges (e.g. USE_CATALOG).
			name: "keeps privileges not implied by ALL_PRIVILEGES",
			input: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{
					catalog.PrivilegeUseCatalog,
					catalog.PrivilegeManage,
					catalog.PrivilegeReadMetadata,
					catalog.PrivilegeExternalUseSchema,
					catalog.PrivilegeExternalUseLocation,
					catalog.PrivilegeAllPrivileges,
				}},
			},
			expected: []catalog.PrivilegeAssignment{
				{Principal: "alice", Privileges: []catalog.Privilege{
					catalog.PrivilegeAllPrivileges,
					catalog.PrivilegeExternalUseLocation,
					catalog.PrivilegeExternalUseSchema,
					catalog.PrivilegeManage,
					catalog.PrivilegeReadMetadata,
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
