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
				Schemas:               map[string]*resources.Schema{"s": {Grants: grants()}},
				FeatureSpecs:          map[string]*resources.FeatureSpec{"f": {Grants: grants()}},
				ModelServices:         map[string]*resources.ModelService{"m": {Grants: grants()}},
				McpServices:           map[string]*resources.McpService{"m": {Grants: grants()}},
				ModelProviderServices: map[string]*resources.ModelProviderService{"m": {Grants: grants()}},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, MergeGrants())
	require.NoError(t, diags.Error())

	merged := []catalog.PrivilegeAssignment{
		{Principal: "alice", Privileges: []catalog.Privilege{catalog.PrivilegeUseSchema, catalog.PrivilegeSelect, catalog.PrivilegeModify}},
		{Principal: "bob", Privileges: []catalog.Privilege{catalog.PrivilegeSelect}},
	}
	assert.Equal(t, merged, b.Config.Resources.Schemas["s"].Grants)
	assert.Equal(t, merged, b.Config.Resources.FeatureSpecs["f"].Grants)
	assert.Equal(t, merged, b.Config.Resources.ModelServices["m"].Grants)
	assert.Equal(t, merged, b.Config.Resources.McpServices["m"].Grants)
	assert.Equal(t, merged, b.Config.Resources.ModelProviderServices["m"].Grants)
}
