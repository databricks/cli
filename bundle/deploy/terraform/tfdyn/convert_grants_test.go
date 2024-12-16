package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertGrants(t *testing.T) {
	src := resources.RegisteredModel{
		Grants: []resources.Grant{
			{
				Privileges: []string{"EXECUTE", "FOO"},
				Principal:  "jane@doe.com",
			},
			{
				Privileges: []string{"EXECUTE", "BAR"},
				Principal:  "spn",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
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

	ctx := context.Background()
	resource := convertGrantsResource(ctx, vin)
	assert.Nil(t, resource)
}

func TestConvertGrantsEmpty(t *testing.T) {
	src := resources.RegisteredModel{
		Grants: []resources.Grant{},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	resource := convertGrantsResource(ctx, vin)
	assert.Nil(t, resource)
}
