package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertRegisteredModel(t *testing.T) {
	src := resources.RegisteredModel{
		CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
			Name:        "name",
			CatalogName: "catalog",
			SchemaName:  "schema",
			Comment:     "comment",
		},
		Grants: []resources.Grant{
			{
				Privileges: []string{"EXECUTE"},
				Principal:  "jane@doe.com",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = registeredModelConverter{}.Convert(ctx, "my_registered_model", vin, out)
	require.NoError(t, err)

	// Assert equality on the registered model
	assert.Equal(t, map[string]any{
		"name":         "name",
		"catalog_name": "catalog",
		"schema_name":  "schema",
		"comment":      "comment",
	}, out.RegisteredModel["my_registered_model"])

	// Assert equality on the grants
	assert.Equal(t, &schema.ResourceGrants{
		Function: "${databricks_registered_model.my_registered_model.id}",
		Grant: []schema.ResourceGrantsGrant{
			{
				Privileges: []string{"EXECUTE"},
				Principal:  "jane@doe.com",
			},
		},
	}, out.Grants["registered_model_my_registered_model"])
}
