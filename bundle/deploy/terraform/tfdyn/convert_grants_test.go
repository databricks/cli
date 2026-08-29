package tfdyn

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertGrants(t *testing.T) {
	src := resources.RegisteredModel{
		Grants: []catalog.PrivilegeAssignment{
			{
				Privileges: []catalog.Privilege{"EXECUTE", "FOO"},
				Principal:  "jane@doe.com",
			},
			{
				Privileges: []catalog.Privilege{"EXECUTE", "BAR"},
				Principal:  "spn",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	resource := convertGrantsResource(ctx, vin)
	require.NotNil(t, resource)
	assert.Equal(t, []schema.ResourceGrantsGrant{
		{
			Privileges: []string{"EXECUTE", "FOO"},
			Principal:  "jane@doe.com",
		},
		{
			Privileges: []string{"EXECUTE", "BAR"},
			Principal:  "spn",
		},
	}, resource.Grant)
}

func TestConvertGrantsNil(t *testing.T) {
	src := resources.RegisteredModel{
		Grants: nil,
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	resource := convertGrantsResource(ctx, vin)
	assert.Nil(t, resource)
}

func TestConvertGrantsEmpty(t *testing.T) {
	src := resources.RegisteredModel{
		Grants: []catalog.PrivilegeAssignment{},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	resource := convertGrantsResource(ctx, vin)
	assert.Nil(t, resource)
}
