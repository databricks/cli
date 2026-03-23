package resourcemutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeGrantsDuplicatePrincipals(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"my_schema": {
						Grants: []catalog.PrivilegeAssignment{
							{
								Principal:  "user@example.com",
								Privileges: []catalog.Privilege{"CREATE_TABLE"},
							},
							{
								Principal:  "user@example.com",
								Privileges: []catalog.Privilege{"CREATE_TABLE"},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeGrants())
	require.NoError(t, diags.Error())

	grants := b.Config.Resources.Schemas["my_schema"].Grants
	assert.Len(t, grants, 1)
	assert.Equal(t, "user@example.com", grants[0].Principal)
	assert.Equal(t, []catalog.Privilege{"CREATE_TABLE"}, grants[0].Privileges)
}

func TestMergeGrantsDuplicatePrivileges(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"my_schema": {
						Grants: []catalog.PrivilegeAssignment{
							{
								Principal:  "user@example.com",
								Privileges: []catalog.Privilege{"CREATE_TABLE", "CREATE_TABLE"},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeGrants())
	require.NoError(t, diags.Error())

	grants := b.Config.Resources.Schemas["my_schema"].Grants
	assert.Len(t, grants, 1)
	assert.Equal(t, []catalog.Privilege{"CREATE_TABLE"}, grants[0].Privileges)
}

func TestMergeGrantsMergesDifferentPrivileges(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {
						Grants: []catalog.PrivilegeAssignment{
							{
								Principal:  "user@example.com",
								Privileges: []catalog.Privilege{"USE_CATALOG"},
							},
							{
								Principal:  "user@example.com",
								Privileges: []catalog.Privilege{"CREATE_SCHEMA"},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeGrants())
	require.NoError(t, diags.Error())

	grants := b.Config.Resources.Catalogs["my_catalog"].Grants
	assert.Len(t, grants, 1)
	assert.Equal(t, "user@example.com", grants[0].Principal)
	assert.Equal(t, []catalog.Privilege{"USE_CATALOG", "CREATE_SCHEMA"}, grants[0].Privileges)
}

func TestMergeGrantsPreservesDistinctPrincipals(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"my_volume": {
						Grants: []catalog.PrivilegeAssignment{
							{
								Principal:  "user1@example.com",
								Privileges: []catalog.Privilege{"READ_VOLUME"},
							},
							{
								Principal:  "user2@example.com",
								Privileges: []catalog.Privilege{"WRITE_VOLUME"},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeGrants())
	require.NoError(t, diags.Error())

	grants := b.Config.Resources.Volumes["my_volume"].Grants
	assert.Len(t, grants, 2)
	assert.Equal(t, "user1@example.com", grants[0].Principal)
	assert.Equal(t, "user2@example.com", grants[1].Principal)
}

func TestMergeGrantsNoGrants(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"my_schema": {},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeGrants())
	require.NoError(t, diags.Error())
}
