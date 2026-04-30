package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func TestValidateGenieSpacePermissions_NoPermissions(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				GenieSpaces: map[string]*resources.GenieSpace{
					"my_space": {
						GenieSpaceConfig: resources.GenieSpaceConfig{
							Title: "My Space",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateGenieSpacePermissions())
	assert.Empty(t, diags)
}

func TestValidateGenieSpacePermissions_WithPermissionsErrors(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				GenieSpaces: map[string]*resources.GenieSpace{
					"my_space": {
						GenieSpaceConfig: resources.GenieSpaceConfig{
							Title: "My Space",
						},
						Permissions: []resources.Permission{
							{Level: iam.PermissionLevel("CAN_MANAGE"), UserName: "user@example.com"},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateGenieSpacePermissions())
	assert.Len(t, diags, 1)
	assert.Equal(t, "Genie Space permissions are not supported", diags[0].Summary)
}
