package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeGrantsSkippedTypesDeclareGrants(t *testing.T) {
	for _, key := range mergeGrantsSkippedTypes {
		_, ok := config.ResourceTypeAs[resources.GrantsDeclarer](key)
		assert.True(t, ok, "%q is skipped by MergeGrants but does not declare grants", key)
	}
}

func TestMergeGrants(t *testing.T) {
	grants := func() []catalog.PrivilegeAssignment {
		return []catalog.PrivilegeAssignment{
			{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeUseSchema, catalog.PrivilegeSelect}},
			{Principal: "bob", Privileges: []catalog.Privilege{catalog.PrivilegeSelect}},
			{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeSelect, catalog.PrivilegeModify}},
		}
	}
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Schemas:      map[string]*resources.Schema{"s": {Grants: grants()}},
				FeatureSpecs: map[string]*resources.FeatureSpec{"f": {Grants: grants()}},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, MergeGrants())
	require.NoError(t, diags.Error())

	assert.Equal(t, []catalog.PrivilegeAssignment{
		{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeUseSchema, catalog.PrivilegeSelect, catalog.PrivilegeModify}},
		{Principal: "bob", Privileges: []catalog.Privilege{catalog.PrivilegeSelect}},
	}, b.Config.Resources.Schemas["s"].Grants)

	// See mergeGrantsSkippedTypes.
	assert.Equal(t, grants(), b.Config.Resources.FeatureSpecs["f"].Grants)
}
